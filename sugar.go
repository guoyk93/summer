package summer

func Bind[T any](c Context) (o T) {
	c.Bind(&o)
	return
}
