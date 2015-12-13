package stream

type FSMv0 interface {
	Next(Streamv0) (FSMv0, Streamv0)
}

type FSM interface {
}
