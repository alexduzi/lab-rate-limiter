package limiter

type RateLimiter interface {
	Allow() bool
}

type RateLimiterImpl struct {
}

func NewRateLimiter() *RateLimiterImpl {
	return &RateLimiterImpl{}
}

func (r *RateLimiterImpl) Allow() bool {
	return true
}
