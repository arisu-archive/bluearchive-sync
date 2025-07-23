package javax

// Integer represents java.lang.Integer
type Integer struct {
	super Number `javaio:"-"`
	Value int32  `javaio:"value"`
}

func (Integer) ClassName() string {
	return "java.lang.Integer"
}

func (Integer) SerialVersionUID() int64 {
	return 1360826667806852920 // From the parsed data
}

func (i *Integer) Super() any {
	return &i.super
}
