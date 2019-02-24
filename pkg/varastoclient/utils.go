package varastoclient

func panicIfError(err error) {
	if err != nil {
		panic(err)
	}
}
