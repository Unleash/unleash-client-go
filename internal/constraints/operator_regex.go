package constraints

import (
	"regexp"
	"sync"

	lru "github.com/hashicorp/golang-lru/v2"

	"github.com/Unleash/unleash-go-sdk/v6/api"
	"github.com/Unleash/unleash-go-sdk/v6/context"
)

type regexKey struct {
	pattern string
	ci      bool
}

var (
	regexCache     *lru.Cache[regexKey, *regexp.Regexp]
	regexCacheOnce sync.Once
)

func getRegexCache() *lru.Cache[regexKey, *regexp.Regexp] {
	regexCacheOnce.Do(func() {
		cache, _ := lru.New[regexKey, *regexp.Regexp](100)
		regexCache = cache
	})
	return regexCache
}

func operatorRegex(ctx *context.Context, constraint api.Constraint) bool {
	contextValue := ctx.Field(constraint.ContextName)

	key := regexKey{
		pattern: constraint.Value,
		ci:      constraint.CaseInsensitive,
	}

	cache := getRegexCache()

	if re, ok := cache.Get(key); ok {
		return re.MatchString(contextValue)
	}

	pattern := key.pattern
	if key.ci {
		pattern = "(?i:" + pattern + ")"
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return false
	}

	cache.Add(key, re)

	return re.MatchString(contextValue)
}
