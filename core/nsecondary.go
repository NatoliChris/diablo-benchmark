package core


import (
	"fmt"
	"net"
	"sort"
	"sync"
	"time"
)


const (
	WARNING_COOLDOWN float64 = 1.0
)


type Nsecondary struct {
	connectAddr  string
	env          []string
	tags         []string
	systemMap    map[string]BlockchainInterface
}

func NewNsecondary(primaryAddr string, primaryPort int, env, tags []string, systemMap map[string]BlockchainInterface) *Nsecondary {
	var addr string = fmt.Sprintf("%s:%d", primaryAddr, primaryPort)

	return &Nsecondary{
		connectAddr: addr,
		env: env,
		tags: tags,
		systemMap: systemMap,
	}
}

func (this *Nsecondary) Run() error {
	var conn net.Conn
	var rt *runtime
	var err error

	Debugf("connect to primary on tcp address: %s", this.connectAddr)
	conn, err = net.Dial("tcp", this.connectAddr)
	if err != nil {
		return fmt.Errorf("cannot connect to tcp %s: %s",
			this.connectAddr, err.Error())
	} else {
		Debugf("connected to primary on tcp address: %s",
			conn.RemoteAddr().String())
	}

	rt, err = newRuntime(conn, this.env, this.tags, this.systemMap)
	if err != nil {
		return err
	} else {
		defer rt.Close()
	}

	err = rt.prepare()
	if err != nil {
		return err
	}

	err = rt.run()
	if err != nil {
		return err
	}

	return nil
}


type runtime struct {
	conn           *primaryConn
	params         *msgPrimaryParameters
	chainEnv       []string
	chain          BlockchainInterface
	clients        map[int]*runtimeClient
	start          time.Time
	lastSkewWarn   time.Time
	interactions   []*runtimeInteraction

	lock           sync.Mutex
	lastDelayWarn  time.Time
}

func newRuntime(conn net.Conn, env, tags []string, systemMap map[string]BlockchainInterface) (*runtime, error) {
	var this runtime
	var err error
	var ok bool

	this.conn = newPrimaryConn(conn)
	this.chainEnv = env

	this.params, err = this.conn.init(&msgSecondaryParameters{
		tags: tags,
	})

	if err != nil {
		return nil, err
	}

	this.chain, ok = systemMap[this.params.sysname]
	if !ok {
		return nil, fmt.Errorf("unknown interface '%s'",
			this.params.sysname)
	}

	Debugf("use interface '%s'", this.params.sysname)

	this.clients = make(map[int]*runtimeClient, 0)
	this.interactions = make([]*runtimeInteraction, 0)

	return &this, nil
}

func (this *runtime) run() error {
	var interaction *runtimeInteraction
	var now, nextTime, delta float64
	var msg *msgStart
	var err error

	Tracef("sort interactions")
	sort.SliceStable(this.interactions, func(i, j int) bool {
		return (this.interactions[i].schedTime <
			this.interactions[j].schedTime)
	})

	Debugf("synchronize with primary")
	err = this.conn.syncReady()
	if err != nil {
		return err
	}

	Tracef("wait for primary start signal")
	msg, err = this.conn.waitStart()
	if err != nil {
		return err
	}

	Infof("run benchmark for %.3f seconds", msg.duration)
	this.start = time.Now()
	this.lastSkewWarn = this.start
	this.lastDelayWarn = this.start
	now = time.Now().Sub(this.start).Seconds()

	for _, interaction = range this.interactions {
		nextTime = interaction.schedTime
		delta = nextTime - now

		if delta > 0 {
			time.Sleep(time.Duration(delta * float64(time.Second)))
		} else if -delta >= this.params.maxSkew {
			this.warnSkew(-delta)
		}

		now = time.Now().Sub(this.start).Seconds()

		go interaction.trigger()
	}

	Infof("stop benchmark")
	return this.stop()
}

func (this *runtime) stop() error {
	var interaction *runtimeInteraction
	var msg msgResultInteraction
	var elapsed time.Duration
	var err error
	var i, n int

	n = len(this.interactions)

	Debugf("send %d results to primary", n)

	for i, interaction = range this.interactions {
		msg.index = interaction.client.id
		msg.ikind = interaction.ikind

		interaction.lock.Lock()

		if interaction.submitted {
			elapsed = interaction.submitTime.Sub(this.start)
			msg.submitTime = elapsed.Seconds()
		} else {
			msg.submitTime = -1
		}

		if interaction.committed {
			elapsed = interaction.commitTime.Sub(this.start)
			msg.commitTime = elapsed.Seconds()
		} else {
			msg.commitTime = -1
		}

		if interaction.aborted {
			elapsed = interaction.abortTime.Sub(this.start)
			msg.abortTime = elapsed.Seconds()
		} else {
			msg.abortTime = -1
		}

		if interaction.done {
			msg.hasError = (interaction.err != nil)
		} else {
			msg.hasError = false
		}

		interaction.lock.Unlock()

		Tracef("push result %d/%d", i + 1, n)
		err = this.conn.pushResult(&msg)
		if err != nil {
			return err
		}
	}
	
	return this.conn.pushResult(&msgResultDone{})
}

type decodeResult struct {
	decoded  *runtimeInteraction
	err      error
}

func (this *runtime) prepare() error {
	var msgInteraction *msgPrepareInteraction
	var decodeChannel chan *decodeResult
	var msgClient *msgPrepareClient
	var decodeRes *decodeResult
	var numDecoded int
	var msg msgPrepare
	var err error
	var ok bool

	Debugf("prepare runtime")

	numDecoded = 0
	decodeChannel = make(chan *decodeResult)
	defer close(decodeChannel)

	for {
		msg, err = this.conn.waitPrepare()
		if err != nil {
			return err
		}

		_, ok = msg.(*msgPrepareDone)
		if ok {
			Debugf("runtime is ready")
			break
		}

		msgClient, ok = msg.(*msgPrepareClient)
		if ok {
			err = this.prepareClient(msgClient)
			if err != nil {
				return err
			}

			continue
		}

		msgInteraction, ok = msg.(*msgPrepareInteraction)
		if ok {
			err = this.prepareInteraction(msgInteraction,
				decodeChannel)

			if err != nil {
				return err
			} else {
				numDecoded += 1
			}

			continue
		}

		return fmt.Errorf("not implemented prepare message %v", msg)
	}

	for numDecoded > 0 {
		decodeRes = <- decodeChannel

		if decodeRes.err != nil {
			return decodeRes.err
		}

		this.interactions = append(this.interactions,
			decodeRes.decoded)

		numDecoded -= 1
	}

	return nil
}

func (this *runtime) prepareClient(msg *msgPrepareClient) error {
	var inner BlockchainClient
	var logger Logger
	var err error

	Tracef("create client %d", msg.index)

	logger = ExtendLogger(fmt.Sprintf("client[%d]", msg.index))

	inner, err = this.chain.Client(this.params.chainParams, this.chainEnv,
		msg.view, logger)

	if err != nil {
		return err
	}

	this.clients[msg.index] =
		newRuntimeClient(this, msg.index, logger, inner)

	return nil
}

func (this *runtime) prepareInteraction(msg *msgPrepareInteraction, decodeChannel chan<- *decodeResult) error {
	var decoded *runtimeInteraction
	var client *runtimeClient
	var err error
	var ok bool

	Tracef("decode interaction for time %.3f on client %d", msg.time,
		msg.index)

	client, ok = this.clients[msg.index]
	if !ok {
		return fmt.Errorf("invalid client index %d", msg.index)
	}

	go func() {
		decoded, err = decodeInteraction(client, msg)
		decodeChannel <- &decodeResult{ decoded, err }
	}()

	return nil
}

func decodeInteraction(client *runtimeClient, msg *msgPrepareInteraction) (*runtimeInteraction, error) {
	var opaque interface{}
	var err error

	opaque, err = client.decode(msg.payload)
	if err != nil {
		return nil, err
	}

	return newRuntimeInteraction(msg.time, client, msg.ikind, opaque), nil
}

func (this *runtime) warnDelay(iact *runtimeInteraction, delay float64) {
	var now time.Time = time.Now()

	this.lock.Lock()

	if now.Sub(this.lastDelayWarn) < (1 * time.Second) {
		this.lock.Unlock()
		return
	}

	this.lastDelayWarn = now

	this.lock.Unlock()

	Warnf("client %d submits %.3f seconds late", iact.client.id, delay)
}

func (this *runtime) warnSkew(delay float64) {
	var now time.Time = time.Now()

	if now.Sub(this.lastSkewWarn) < (1 * time.Second) {
		return
	}

	Warnf("benchmark is behind schedule by %.3f seconds", delay)
	this.lastSkewWarn = now
}

func (this *runtime) errorInteraction(err error) {
	Errorf("%s", err.Error())
}

func (this *runtime) Close() error {
	Debugf("close connection to primary")
	return this.conn.Close()
}


type runtimeClient struct {
	rt      *runtime
	id      int
	logger  Logger
	inner   BlockchainClient
}

func newRuntimeClient(runtime *runtime, id int, logger Logger, inner BlockchainClient) *runtimeClient {
	return &runtimeClient{
		rt: runtime,
		id: id,
		logger: logger,
		inner: inner,
	}
}

func (this *runtimeClient) trigger(iact Interaction) error {
	return this.inner.TriggerInteraction(iact)
}

func (this *runtimeClient) decode(payload []byte) (interface{}, error) {
	return this.inner.DecodePayload(payload)
}

func (this *runtimeClient) runtime() *runtime {
	return this.rt
}


type runtimeInteraction struct {
	client       *runtimeClient
	ikind        int
	schedTime    float64
	opaque       interface{}

	lock         sync.Mutex
	submitted    bool
	committed    bool
	aborted      bool
	done         bool
	submitTime   time.Time
	commitTime   time.Time
	abortTime    time.Time
	err          error
}

func newRuntimeInteraction(schedTime float64, client *runtimeClient, ikind int, opaque interface{}) *runtimeInteraction {
	var this runtimeInteraction

	this.client = client
	this.ikind = ikind
	this.schedTime = schedTime
	this.opaque = opaque
	this.submitted = false
	this.committed = false
	this.aborted = false
	this.done = false

	return &this
}

func (this *runtimeInteraction) trigger() {
	var err error

	err = this.client.trigger(this)

	this.lock.Lock()

	this.err = err
	this.done = true

	this.lock.Unlock()

	if err != nil {
		this.runtime().errorInteraction(err)
	}
}

func (this *runtimeInteraction) runtime() *runtime {
	return this.client.runtime()
}

func (this *runtimeInteraction) Payload() interface{} {
	return this.opaque
}

func (this *runtimeInteraction) ReportSubmit() {
	var submitTime time.Time
	var delay float64
	var rt *runtime

	submitTime = time.Now()

	this.lock.Lock()

	if this.submitted {
		this.lock.Unlock()
		Warnf("interaction submitted more than once")
		return
	}

	this.submitTime = submitTime
	this.submitted = true

	this.lock.Unlock()

	rt = this.runtime()
	delay = submitTime.Sub(rt.start).Seconds() - this.schedTime

	if delay >= rt.params.maxDelay {
		rt.warnDelay(this, delay)
	}
}

func (this *runtimeInteraction) ReportCommit() {
	this.lock.Lock()

	if this.committed {
		this.lock.Unlock()
		Warnf("interaction committed more than once")
		return
	}

	this.commitTime = time.Now()
	this.committed = true

	this.lock.Unlock()
}

func (this *runtimeInteraction) ReportAbort() {
	this.lock.Lock()

	if this.aborted {
		this.lock.Unlock()
		Warnf("interaction aborted more than once")
		return
	}

	this.abortTime = time.Now()
	this.aborted = true

	this.lock.Unlock()
}
