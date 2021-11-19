package workload


// A workload transaction is an opaque byte array to send to the secondaries
type workloadTransaction []byte

// A workload interval is a list of transaction to send during a time interval
type workloadInterval struct {
	array  []workloadTransaction
}

// A thread workload is the list of workload intervals to send
type ThreadWorkload struct {
	array  []workloadInterval
}

// A secondary workload is the list of the workloads to send for all the
// threads of the secondary
type secondaryWorkload struct {
	array  []ThreadWorkload
}

// A complete workload is the list of the workloads to send for all the
// secondaries
type Workload struct {
	array  []secondaryWorkload
}


func New() *Workload {
	return &Workload{ array: make([]secondaryWorkload, 0) }
}

func (this *Workload) BuildFlat() [][][][][]byte {
	var secondaryId, threadId, interval, index int
	var secondaryWl *secondaryWorkload
	var wlInterval *workloadInterval
	var wlTx workloadTransaction
	var threadWl *ThreadWorkload
	var ret [][][][][]byte

	ret = make([][][][][]byte, len(this.array))
	for secondaryId = range this.array {

		secondaryWl = &this.array[secondaryId]
		ret[secondaryId] = make([][][][]byte, len(secondaryWl.array))

		for threadId = range secondaryWl.array {

			threadWl = &secondaryWl.array[threadId]
			ret[secondaryId][threadId] = make([][][]byte,
				len(threadWl.array))

			for interval = range threadWl.array {

				wlInterval = &threadWl.array[interval]
				ret[secondaryId][threadId][interval] =
					make([][]byte, len(wlInterval.array))

				for index, wlTx = range wlInterval.array {

					ret[secondaryId][threadId][interval][index] = wlTx

				}
			}
		}
	}

	return ret
}


func (this *workloadInterval) add(tx []byte) {
	this.array = append(this.array, tx)
}


func (this *ThreadWorkload) getInterval(index int) *workloadInterval {
	for len(this.array) <= index {
		this.array = append(this.array, workloadInterval{
			array: make([]workloadTransaction, 0),
		})
	}

	return &this.array[index]
}

func (this *ThreadWorkload) add(interval int, tx []byte) {
	this.getInterval(interval).add(tx)
}


func (this *secondaryWorkload) getThread(index int) *ThreadWorkload {
	for len(this.array) <= index {
		this.array = append(this.array, ThreadWorkload{
			array: make([]workloadInterval, 0),
		})
	}

	return &this.array[index]
}

func (this *secondaryWorkload) add(threadId, interval int, tx []byte) {
	this.getThread(threadId).add(interval, tx)
}


func (this *Workload) getSecondary(index int) *secondaryWorkload {
	for len(this.array) <= index {
		this.array = append(this.array, secondaryWorkload{
			array: make([]ThreadWorkload, 0),
		})
	}

	return &this.array[index]
}

func (this *Workload) Add(secondaryId, threadId, interval int, tx []byte) {
	this.getSecondary(secondaryId).add(threadId, interval, tx)
}
