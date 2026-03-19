# Shimmy: A Design Manifesto
### Principled Decorator Generation for Go Interfaces

---

## The Problem

Go's interfaces are one of its finest features. They are implicit, composable, and define clean contracts between components. But Go is deliberately primitive when it comes to *wrapping* those interfaces with cross-cutting behavior — things like logging, metrics, caching, tracing, and retry logic.

In languages like TypeScript, this kind of thing is approachable through mixins, decorators, and higher-order types. In Go, the idiomatic answer is the **decorator pattern**: write a struct that holds an inner implementation, implements the same interface, and delegates each method call while adding behavior around it.

The problem is that this is pure boilerplate. For an interface with eight methods, you write eight wrapper functions that all look structurally identical. For a logging decorator, a caching decorator, and a metrics decorator, you write twenty-four. Most of them are near-identical scaffolding surrounding a single line of actual logic.

There are existing tools — `gowrap`, `wrapgen`, `go-decorator` — that address the code generation side of this. But they share a common limitation: they generate *static* wrappers. You get a fully written `LoggedUserRepository`, but it can't be composed with other decorators, it doesn't support per-method hooks, and it offers no way to intercept a call uniformly across all methods without giving up type information.

Shimmy is an attempt to design a better answer from first principles.

---

## What We Actually Want

Before arriving at a design, it helps to be precise about the use cases.

**Use case 1: Uniform cross-cutting behavior.**
I want to add structured logging to every method of `UserRepository`. Every call should log its method name, its arguments, and its return values. I should not have to write this twelve times. The behavior is identical for every method.

**Use case 2: Method-specific behavior with early return.**
I want to add caching to `GetUserProfile` specifically. When the cache has a hit, I want to return immediately without calling the underlying implementation. For every other method, I want no caching behavior at all. The behavior is specific to one method and it needs to short-circuit the call.

**Use case 3: Composing multiple behaviors.**
I want both. A `LoggedCachedUserRepository` that logs every call *and* caches `GetUserProfile`. I want to compose these two behaviors without re-implementing either, and without coupling them to each other.

Any design that handles all three cleanly is worth pursuing.

---

## The Evolution of the Design

### First attempt: hand-written decorators

The naive Go answer is to write it by hand.

```go
type LoggedUserRepository struct {
    inner  UserRepository
    logger *slog.Logger
}

func (r *LoggedUserRepository) GetUserProfile(ctx context.Context, id user.UserId) (*UserProfile, error) {
    r.logger.Info("GetUserProfile called", "id", id)
    result, err := r.inner.GetUserProfile(ctx, id)
    r.logger.Info("GetUserProfile returned", "err", err)
    return result, err
}

// ... seven more methods
```

This works but doesn't scale. It's not reusable across different interfaces. Every new interface requires another full decorator struct. The actual logic (the two log lines) is buried in identical scaffolding.

### Second attempt: code generation with static templates

Tools like `gowrap` solve the repetition by generating the scaffolding from a template. You define what a logging wrapper looks like once, and the tool produces the concrete implementation for any interface.

This is better, but the generated code is still static. You can't compose two generated decorators without writing a third. You can't configure behavior per-method. The template is applied uniformly and the result is inflexible.

### Third attempt: a generated shim with hook fields

The key insight is to generate a *generic intermediary* — a shim — rather than a specific decorator. The shim itself contains no behavior. It holds the inner implementation and exposes hook points that the caller configures. The generated code handles delegation; the user provides only the logic that actually differs.

The natural hook shape is `Around`: a function that receives the method arguments, a `next` callable representing the inner implementation, and returns the method's return values. This single shape covers before-hooks, after-hooks, and early returns uniformly — a before-hook is an Around that always calls next, an early return is an Around that sometimes doesn't.

```go
type UserRepositoryShim struct {
    Inner                UserRepository
    AroundGetUserProfile func(id user.UserId, next func(user.UserId) (*UserProfile, error)) (*UserProfile, error)
    // ...
}
```

This is expressive. But it has a composition problem. Each method accepts only one Around function. Stacking logging and caching on the same method requires manually composing two closures, which produces deeply nested, type-heavy code that is hard to read and harder to extend.

### Fourth attempt: middleware + per-method overrides

The composition problem is solved by separating the two levels of granularity into distinct mechanisms.

The shim supports a slice of **middleware** — each middleware is a struct that can implement Around hooks for any subset of the interface's methods. A middleware that implements a hook for `GetUserProfile` handles that method; it passes through every other method untouched. Multiple middleware are chained in order.

For cases requiring surgical precision, the shim also accepts **per-method Around functions** that run after the middleware chain. These are the escape hatch for one-off behavior that doesn't need to be packaged as a reusable middleware.

The boilerplate that makes middleware composable — the `Base` embedding struct, the chaining logic — is entirely generated. The user only writes the logic they care about.

```go
// LoggingMiddleware only overrides the methods it cares about.
// All others pass through via the embedded Base.
type LoggingMiddleware struct {
    BaseUserRepositoryMiddleware
    logger *slog.Logger
}

func (m *LoggingMiddleware) AroundGetUserProfile(
    id user.UserId,
    next func(user.UserId) (*UserProfile, error),
) (*UserProfile, error) {
    m.logger.Info("GetUserProfile called", "id", id)
    result, err := next(id)
    m.logger.Info("GetUserProfile returned", "err", err)
    return result, err
}
```

Composing logging and caching is now just:

```go
shim := &UserRepositoryShim{
    Inner: pgRepo,
    Middleware: []UserRepositoryMiddleware{
        &LoggingMiddleware{logger: log},
        &CacheMiddleware{cache: c},
    },
}
```

This is clean. But there is still a gap: truly uniform behavior — something that applies to every method identically, without the user overriding every method individually — is still awkward. For a `LoggingMiddleware` that logs every call in the same way, you still need to write N method overrides that all look the same.

### Fifth attempt: the Interceptor and the Call envelope

The remaining gap is uniform, typeless observation of every call. The solution is a second hook mechanism: an **Interceptor**, which fires for every method invocation before the middleware chain.

The challenge with an Interceptor is that it receives calls to methods with different signatures. Without reflection, you cannot have a single function that receives the arguments to `GetUserProfile` and also the arguments to `CreateUser`. The naive solution — `invoke func()` with no arguments — is useless, because the caller can't see what went in or came out.

The solution is a **Call envelope**: a generated struct per method that carries both inputs and outputs.

```go
// Generated by shimmy
type GetUserProfileCall struct {
    // Inputs — populated before invoke
    Ctx    context.Context
    UserId user.UserId
    // Outputs — populated by invoke
    Result *UserProfile
    Err    error
}

func (c *GetUserProfileCall) Method() string { return "GetUserProfile" }
```

The Interceptor receives any `Call` and an `invoke` function. Before `invoke` is called, the Call struct contains only inputs. After `invoke` returns, it contains both. The Call struct is mutated in place, so the Interceptor sees the full picture with a single argument.

```go
type UserRepositoryShim struct {
    Inner       UserRepository
    Middleware  []UserRepositoryMiddleware
    Interceptor func(call shimmy.Call, invoke func())
}
```

For uniform behavior, no type assertion is needed:

```go
shim.Interceptor = func(call shimmy.Call, invoke func()) {
    start := time.Now()
    invoke()
    metrics.Record(call.Method(), time.Since(start))
}
```

For surgical access to specific arguments:

```go
shim.Interceptor = func(call shimmy.Call, invoke func()) {
    invoke()
    if c, ok := call.(*GetUserProfileCall); ok && c.Err != nil {
        alerting.Notify("profile lookup failed", "userId", c.UserId, "err", c.Err)
    }
}
```

The type assertion is the opt-in. You reach for it only when you need it. When you don't, the Interceptor is fully generic.

But this design has two remaining problems. First, the Interceptor is a special-cased field on the shim, separate from middleware — which raises a confusing question for users: what is the difference, and which should I use? Second, the Interceptor always fires before the entire middleware chain, so its position relative to other behaviors is fixed and cannot be controlled. If you want metrics to record only the time spent in the cache layer and not in the logging layer, you cannot express that.

### Sixth attempt: Interceptors as first-class middleware

The key realization is that an Interceptor is not a fundamentally different concept from a middleware — it is just a middleware that operates uniformly across all methods, without needing to name each one. The distinction is not in kind but in granularity. Treating the Interceptor as a separate field on the shim was a premature specialization.

The fix is to collapse both concepts into the single `Middleware` slice and let users place Interceptor-style behavior anywhere in the chain by wrapping it with a generated constructor.

The generator emits a `NewUserRepositoryInterceptor` constructor alongside everything else. It accepts a `func(shimmy.Call, invoke func())` and returns a `UserRepositoryMiddleware`. Internally, the generated implementation has an Around method for every interface method; each one constructs the appropriate Call envelope, wraps `next` as `invoke func()`, and delegates to the user's function. From the outside it is indistinguishable from any other middleware — it sits in the slice, it respects position, it can short-circuit.

```go
// Generated by shimmy
type userRepositoryInterceptor struct {
    fn func(shimmy.Call, func())
}

func NewUserRepositoryInterceptor(fn func(shimmy.Call, func())) UserRepositoryMiddleware {
    return &userRepositoryInterceptor{fn: fn}
}

func (i *userRepositoryInterceptor) AroundGetUserProfile(
    ctx context.Context, id user.UserId,
    next func(context.Context, user.UserId) (*UserProfile, error),
) (*UserProfile, error) {
    call := &GetUserProfileCall{Ctx: ctx, Id: id}
    invoke := func() {
        call.Result, call.Err = next(call.Ctx, call.Id)
    }
    i.fn(call, invoke)
    return call.Result, call.Err
}

// ... same pattern for every method
```

To support uniform argument and result access without type assertions, the Call envelope is extended with `Args()` and `Results()` accessors, also generated:

```go
func (c *GetUserProfileCall) Args() []any    { return []any{c.Ctx, c.Id} }
func (c *GetUserProfileCall) Results() []any { return []any{c.Result, c.Err} }
```

The `shimmy.Call` interface in the runtime package becomes:

```go
type Call interface {
    Method()   string
    Args()     []any
    Results()  []any
}
```

Now uniform logging across every method — the original motivating use case — requires no type assertion and no per-method boilerplate:

```go
NewUserRepositoryInterceptor(func(call shimmy.Call, invoke func()) {
    log.Info("→", "method", call.Method(), "args", call.Args())
    invoke()
    log.Info("←", "method", call.Method(), "results", call.Results())
})
```

And composing it with typed middleware is just a matter of slice position:

```go
shim := &UserRepositoryShim{
    Inner: postgresRepo,
    Middleware: []UserRepositoryMiddleware{
        &UserRepositoryLogging{logger: log},
        NewUserRepositoryInterceptor(func(call shimmy.Call, invoke func()) {
            start := time.Now()
            invoke()
            metrics.RecordLatency(call.Method(), time.Since(start))
        }),
        &UserRepositoryCache{cache: redisCache},
    },
}
```

The per-method escape hatch fields (`AroundGetUserProfile func(...)` on the shim struct) are dropped entirely. They were the path of least resistance that users would reach for by default, bypassing the composable middleware model. Everything is now expressed through the `Middleware` slice — either as a typed middleware struct or as a positioned `NewUserRepositoryInterceptor` call. The two mechanisms are unified, the shim struct is simpler, and the mental model for users has one fewer concept.

---

## The Final Design

Shimmy generates five things from an interface definition:

**1. A shim struct.** Holds the inner implementation and a middleware slice. No special-cased fields; all behavior is expressed through middleware.

**2. A middleware interface.** One Around method per interface method, with full type signatures. Implementing the interface is the contract for a middleware.

**3. A Base middleware struct.** Implements every method in the middleware interface as a pure passthrough. Middleware authors embed this and override only the methods they care about.

**4. A Call envelope per method.** A generated struct carrying all inputs and outputs for a single method invocation, implementing the `shimmy.Call` interface. Exposes `Method()`, `Args()`, and `Results()` for generic access, plus typed fields for surgical access via type assertion.

**5. A `NewXxxInterceptor` constructor.** Accepts a `func(shimmy.Call, func())` and returns an `XxxMiddleware`. Allows uniform, position-controlled behavior to be inserted anywhere in the middleware chain without writing per-method overrides.

The execution order for any method invocation is simply the middleware slice, in order:

```
Middleware[0].AroundX
  → Middleware[1].AroundX
    → Middleware[2].AroundX
      → ...
        → Inner.X()
```

Each layer controls whether the next layer is called. Any layer may short-circuit and return early. Interceptor-style middleware participates in this chain like any other middleware.

The `shimmy` runtime package itself stays minimal: the `Call` interface definition and nothing else. All generated types live in the user's package.

---

## Naming Conventions

Middleware implementations should be named `<Interface><Behavior>`, not `<Behavior>Middleware`. The `Middleware` suffix is redundant since the type is already constrained to `<Interface>Middleware` at the call site. Concrete examples:

- `UserRepositoryLogging` not `LoggingMiddleware`
- `UserRepositoryCache` not `CacheMiddleware`
- `UserRepositoryRetry` not `RetryMiddleware`

This makes the interface affiliation explicit and avoids implying that the behavior is generic across all interfaces when it is not.

---

## What Shimmy Does Not Do

**It does not use reflection.** The generated code is fully typed. The only place a type assertion appears is in user-written interceptor code that explicitly opts into method-specific argument access.

**It does not modify your interface.** The interface is the contract and it stays untouched. Shimmy observes it during code generation but adds no shimmy-specific types to the interface's signatures or its implementations.

**It does not enforce a specific behavior.** Shimmy generates the scaffolding. What you put in the hooks is entirely up to you.

---

## Design Principles, Summarized

1. **Interfaces are sacred.** The domain interface is the contract. It must not be contaminated with infrastructure types.

2. **Boilerplate is generated; logic is written.** The user writes the part that differs. The generator writes everything else.

3. **Composition over monoliths.** Behaviors are separate, stackable, independently testable units. Combining them requires no glue code.

4. **Type safety by default, type assertions by choice.** The only place the type system loosens is in interceptor code, and only when the caller explicitly opts in via a type assertion.

5. **Around is the universal shape.** Before-hooks, after-hooks, and early returns are all degenerate cases of Around. A single shape for all hook types reduces the API surface area and eliminates conceptual overhead.

6. **One mechanism, not two.** Interceptors and middleware are not distinct concepts. An interceptor is middleware that operates uniformly. Unifying them into the slice eliminates a confusing conceptual split and gives users full control over ordering.

---

## Open Questions

- **`Args()` and context.** Should `Args()` include the `context.Context` argument? For logging it is usually noise. The generator could omit it from `Args()` and expose it as a separate `Context()` accessor, or generate both `Args()` and `ArgsWithContext()`. A minor detail but worth a decision before the generator is written.

- **Call struct mutation and concurrency.** The Call struct is mutated in place by `invoke`. If user code spawns goroutines inside an interceptor function (e.g., async metrics emission), the mutation is a data race. The design does not currently prevent this. Worth documenting as a constraint.

- **Error transformation.** A common middleware use case is translating infrastructure errors (Postgres errors, cache misses) into domain errors. This fits naturally into the Around pattern, but may benefit from a first-class helper.

- **Middleware chains across interfaces.** If two interfaces share common method shapes (e.g., both have a `ctx context.Context` first argument), could a single middleware apply to both? This may be out of scope for v1 but is worth considering as an extension.

- **CLI and generation interface.** The tool needs a clean invocation model — likely directive comments or a config file pointing at interface definitions — that integrates well with `go generate`. Edge cases around embedded interfaces, interfaces defined in external packages, and interfaces with generic type parameters need to be scoped before the generator is written.
