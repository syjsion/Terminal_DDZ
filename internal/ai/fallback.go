package ai

import "fmt"

type FallbackError struct {
	Provider string
	Cause    error
}

func (e *FallbackError) Error() string {
	return fmt.Sprintf("%s 请求失败，已切换本地 AI: %v", e.Provider, e.Cause)
}

func (e *FallbackError) Unwrap() error { return e.Cause }
