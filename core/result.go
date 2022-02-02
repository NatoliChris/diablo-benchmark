package core


type Result struct {
	Seed       int64
	Locations  []*SecondaryResult
}

type SecondaryResult struct {
	Address  string
	Tags     []string
	ids      map[int]int
	Clients  []*ClientResult
}

type ClientResult struct {
	Index         int
	Kind          string
	Interactions  []*InteractionResult
}

type InteractionResult struct {
	Kind        string
	SubmitTime  float64  // negative if not submitted
	CommitTime  float64  // negative if not committed
	AbortTime   float64  // negative if not aborted
	HasError    bool
}


func newResult(masterSeed int64) *Result {
	return &Result{
		Seed: masterSeed,
		Locations: make([]*SecondaryResult, 0),
	}
}

func (this *Result) addSecondary(sresult *SecondaryResult) {
	this.Locations = append(this.Locations, sresult)
}


func newSecondaryResult(addr string, tags []string) *SecondaryResult {
	return &SecondaryResult{
		Address: addr,
		Tags: tags,
		ids: make(map[int]int, 0),
		Clients: make([]*ClientResult, 0),
	}
}

func (this *SecondaryResult) getClientResult(id int, kind string) *ClientResult {
	var offset int
	var ok bool

	offset, ok = this.ids[id]
	if !ok {
		offset = len(this.Clients)
		this.ids[id] = offset
		this.Clients = append(this.Clients, &ClientResult{
			Index: id,
			Kind: kind,
			Interactions: make([]*InteractionResult, 0),
		})
	}

	return this.Clients[offset]
}

func (this *SecondaryResult) addResult(clientId int, clientKind, interactionKind string, submitTime, commitTime, abortTime float64, hasError bool) {
	var client *ClientResult = this.getClientResult(clientId, clientKind)

	client.Interactions = append(client.Interactions, &InteractionResult{
		Kind: interactionKind,
		SubmitTime: submitTime,
		CommitTime: commitTime,
		AbortTime: abortTime,
		HasError: hasError,
	})
}
