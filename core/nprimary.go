package core


import (
	"fmt"
	"net"
)


type Nprimary struct {
	NumSecondary   int

	SetupPath      string

	BenchmarkPath  string

	SystemMap      map[string]BlockchainInterface

	ListenPort     int

	MasterSeed     int64

	MaxSkew        float64

	MaxDelay       float64

	Env            []string
}

func (this *Nprimary) Run() (*Result, error) {
	var duration, secondaryDuration float64
	var secondaries []*remoteSecondary
	var endpoints map[string][]string
	var chain BlockchainInterface
	var builder BlockchainBuilder
	var sresult *SecondaryResult
	var locations []location
	var endpoint endpoint
	var result *Result
	var logger Logger
	var tags []string
	var setup setup
	var sys *system
	var err error
	var ok bool
	var i int

	Debugf("use master seed: %d", this.MasterSeed)
	Debugf("parse setup file '%s'", this.SetupPath)
	setup, err = parseSetupYamlPath(this.SetupPath)
	if err != nil {
		return nil, err
	}

	Debugf("use interface '%s'", setup.sysname())

	endpoints = make(map[string][]string, len(secondaries))
	for _, endpoint = range setup.endpoints() {
		tags = make([]string, len(endpoint.tags()))
		copy(tags, endpoint.tags())
		endpoints[endpoint.address()] = tags
	}

	chain, ok = this.SystemMap[setup.sysname()]
	if !ok {
		return nil,fmt.Errorf("unknown interface '%s'",setup.sysname())
	}

	logger = ExtendLogger("builder")
	builder, err = chain.Builder(setup.parameters(), this.Env,
		endpoints, logger)
	if err != nil {
		return nil, err
	}

	Debugf("wait for %d secondary connections", this.NumSecondary)
	secondaries, err = this.acceptSecondaries(setup)
	if err != nil {
		return nil, err
	}

	locations = make([]location, len(secondaries))
	for i = range secondaries {
		locations[i] = secondaries[i]
	}

	sys = newSystem(this.MasterSeed, locations, setup, builder)

	Debugf("parse benchmark '%s'", this.BenchmarkPath)
	err = parseBenchmarkYamlPath(this.BenchmarkPath, sys)
	if err != nil {
		return nil, err
	}

	duration = 0
	for i = range secondaries {
		secondaryDuration = secondaries[i].end()
		if secondaryDuration > duration {
			duration = secondaryDuration
		}
	}

	Debugf("benchmark duration is %.3f seconds", duration)

	Debugf("synchronize with secondaries")
	for i = range secondaries {
		Tracef("wait for secondary %s", secondaries[i].addr())
		secondaries[i].ready()
	}	

	Infof("start benchmark")
	for i = range secondaries {
		Tracef("send start signal to %s", secondaries[i].addr())
		secondaries[i].start(duration)
	}

	Debugf("end of benchmark")

	result = newResult(this.MasterSeed)
	for i = range secondaries {
		Tracef("collect results from %s", secondaries[i].addr())
		sresult, err = secondaries[i].collect()
		if err != nil {
			return nil, err
		}

		result.addSecondary(sresult)
		secondaries[i].Close()
	}

	return result, nil
}

func (this *Nprimary) acceptSecondaries(setup setup) ([]*remoteSecondary, error) {
	var laddr, raddr, tag string
	var ret []*remoteSecondary
	var listener net.Listener
	var conn net.Conn
	var err error
	var done bool
	var i int

	laddr = fmt.Sprintf("0.0.0.0:%d", this.ListenPort)
	ret = make([]*remoteSecondary, this.NumSecondary)

	Debugf("listen for %d secondary connections on %s", len(ret), laddr)
	listener, err = net.Listen("tcp", laddr)
	if err != nil {
		return nil, err
	}

	done = false

	defer func() {
		Debugf("close listener on %s", laddr)
		listener.Close()

		if done {
			return
		}

		for i = range ret {
			if ret[i] == nil {
				continue
			}

			Debugf("close connection from %s", ret[i].addr())
			ret[i].Close()
		}
	}()

	for i = range ret {
		Tracef("wait for connection on %s", laddr)
		conn, err = listener.Accept()
		if err != nil {
			return nil, err
		}

		raddr = conn.RemoteAddr().String()
		Debugf("new secondary connection from %s", raddr)

		ret[i], err = newRemoteSecondary(conn, setup, this)
		if err != nil {
			conn.Close()
			return nil, err
		}

		Tracef("secondary %s tags:", raddr)
		for _, tag = range ret[i].tags() {
			Tracef("  %s", tag)
		}
	}

	done = true

	return ret, nil
}


type remoteSecondary struct {
	conn     *secondaryConn
	params   *msgSecondaryParameters
	clients  []*remoteClient
}

func newRemoteSecondary(conn net.Conn, setup setup, primary *Nprimary) (*remoteSecondary, error) {
	var this remoteSecondary
	var err error

	this.conn = newSecondaryConn(conn)
	this.clients = make([]*remoteClient, 0)

	this.params, err = this.conn.init(&msgPrimaryParameters{
		sysname: setup.sysname(),
		chainParams: setup.parameters(),
		maxDelay: primary.MaxDelay,
		maxSkew: primary.MaxSkew,
	})

	if err != nil {
		return nil, err
	}

	this.params.tags = append(this.params.tags, this.addr())

	return &this, nil
}

func (this *remoteSecondary) createClient(kind string, view []string) (client, error) {
	var id int = len(this.clients)
	var client *remoteClient
	var err error

	Tracef("prepare client %d (%s) on secondary %s", id, kind, this.addr())
	err = this.conn.sendPrepare(&msgPrepareClient{
		view: view,
		index: id,
	})

	if err != nil {
		return nil, err
	}

	client = newRemoteClient(this.conn, id, kind)
	this.clients = append(this.clients, client)

	return client, nil
}

func (this *remoteSecondary) tags() []string {
	return this.params.tags
}

func (this *remoteSecondary) ready() error {
	return this.conn.syncReady()
}

func (this *remoteSecondary) start(duration float64) error {
	return this.conn.sendStart(&msgStart{
		duration: duration,
	})
}

func (this *remoteSecondary) collect() (*SecondaryResult, error) {
	var msgIact *msgResultInteraction
	var result *SecondaryResult
	var client *remoteClient
	var msg msgResult
	var err error
	var ok bool

	result = newSecondaryResult(this.addr(), this.params.tags)

	for {
		Tracef("pull next result from %s", this.addr())
		msg, err = this.conn.pullResult()
		if err != nil {
			return nil, err
		}

		_, ok = msg.(*msgResultDone)
		if ok {
			break
		}

		msgIact, ok = msg.(*msgResultInteraction)
		if ok {
			Tracef("new interaction result for kind %d on " +
				"client %d for %s", msgIact.ikind,
				msgIact.index, this.addr())

			if msgIact.index >= len(this.clients) {
				return nil, fmt.Errorf("invalid client id " +
					"%d for secondary %s", msgIact.index,
					this.addr())
			}

			client = this.clients[msgIact.index]

			if msgIact.ikind >= len(client.kinds) {
				return nil, fmt.Errorf("invalid interaction " +
					"ikind %d for client %d on " +
					"secondary %s", msgIact.ikind,
					msgIact.index, this.addr())
			}

			result.addResult(msgIact.index, client.kind,
				client.kinds[msgIact.ikind],
				msgIact.submitTime, msgIact.commitTime,
				msgIact.abortTime, msgIact.hasError)

			continue
		}

		return nil, fmt.Errorf("not implemented result message %v",msg)
	}

	Tracef("end of results for %s", this.addr())
	return result, nil
}

func (this *remoteSecondary) end() float64 {
	var clientEnd, maxTime float64
	var client *remoteClient

	maxTime = 0

	for _, client = range this.clients {
		clientEnd = client.end()

		if clientEnd > maxTime {
			maxTime = clientEnd
		}
	}

	return maxTime
}

func (this *remoteSecondary) addr() string {
	return this.conn.addr()
}

func (this *remoteSecondary) Close() error {
	return this.conn.Close()
}


type remoteClient struct {
	conn     *secondaryConn
	index    int
	kind     string
	maxTime  float64
	kinds    []string
	ikinds   map[string]int
}

func newRemoteClient(conn *secondaryConn, index int, kind string) *remoteClient {
	return &remoteClient{
		conn: conn,
		index: index,
		kind: kind,
		maxTime: 0,
		kinds: make([]string, 0),
		ikinds: make(map[string]int, 0),
	}
}

func (this *remoteClient) sendInteraction(kind string, time float64, encoded []byte) error {
	var ikind int
	var ok bool

	if time > this.maxTime {
		this.maxTime = time
	}

	ikind, ok = this.ikinds[kind]
	if !ok {
		ikind = len(this.kinds)
		this.kinds = append(this.kinds, kind)
		this.ikinds[kind] = ikind
	}

	Tracef("prepare transaction %d (%s) for time %.3f on client %d " +
		"secondary %s", ikind, kind, time, this.index,
		this.conn.addr())

	return this.conn.sendPrepare(&msgPrepareInteraction{
		index: this.index,
		ikind: ikind,
		time: time,
		payload: encoded,
	})
}

func (this *remoteClient) end() float64 {
	return this.maxTime
}
