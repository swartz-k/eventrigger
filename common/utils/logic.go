package utils

func RetryWithCount(fn func() error, count int) (err error) {
	for i := 0; i < count; i++ {
		err = fn()
		if err == nil {
			return nil
		}
	}
	return err
}
