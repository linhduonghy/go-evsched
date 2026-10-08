package http

import (
	"fmt"
	"net/http"
	"strconv"
)

func PathInt(r *http.Request, key string) (int, error) {
	v := r.PathValue(key)
	if v == "" {
		return 0, fmt.Errorf("no value for %s", key)
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return 0, err
	}
	return i, nil
}

func PathInt64(r *http.Request, key string) (int64, error) {
	v := r.PathValue(key)
	if v == "" {
		return 0, fmt.Errorf("no value for %s", key)
	}
	i, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return 0, err
	}
	return i, nil
}

func PathString(r *http.Request, key string) string {
	return r.PathValue(key)
}

func QueryInt(r *http.Request, key string, defaultValue int) (int, error) {
	v := r.URL.Query().Get(key)
	if v == "" {
		return defaultValue, nil
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return defaultValue, err
	}
	return i, nil
}

func QueryParam(r *http.Request, key string) string {
	return r.URL.Query().Get(key)
}
