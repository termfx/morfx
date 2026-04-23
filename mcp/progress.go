package mcp

import "context"

type progressContextKey struct{}

type progressState struct {
	token string
}

func withProgressToken(ctx context.Context, token string) context.Context {
	if token == "" {
		return ctx
	}
	state := progressState{token: token}
	return context.WithValue(ctx, progressContextKey{}, state)
}

func progressTokenFromContext(ctx context.Context) (string, bool) {
	if ctx == nil {
		return "", false
	}
	if state, ok := ctx.Value(progressContextKey{}).(progressState); ok {
		if state.token != "" {
			return state.token, true
		}
	}
	return "", false
}
