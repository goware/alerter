package alerter

type Alerter interface {
	Alert(format string, v ...interface{})
}
