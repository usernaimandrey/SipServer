package repositoriy

type CallEndedBy string

const (
	CallEndedByCaller CallEndedBy = "caller"
	CallEndedByCallee CallEndedBy = "callee"
	CallEndedBySystem CallEndedBy = "system"
)
