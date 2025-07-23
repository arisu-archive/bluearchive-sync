package javax

// Number represents java.lang.Number
type Number struct{}

func (Number) ClassName() string {
	return "java.lang.Number"
}

func (Number) SerialVersionUID() int64 {
	return -8742448824652078965 // From the parsed data
}
