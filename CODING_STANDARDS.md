# Coding standards

These are initial project design and review conventions, not settled game rules.

- Use the project's agreed domain vocabulary in exported behavior and tests.
- Keep a small public interface around substantial cohesive behavior. Introduce
  abstractions when an actual boundary needs them, not for hypothetical extensions.
- Keep game rules independent of transport and storage. Pass time and randomness
  through explicit boundaries when their effects need deterministic verification.
- Represent chip quantities exactly with integers; validate amounts and arithmetic.
  A denomination, maximum balance, and settlement policy belong in the spec.
- Server state determines legal actions and each recipient's visible information.
  Treat client commands as untrusted inputs; expose only the agreed player view.
- Make ownership of mutable table state explicit. Choose and document its concurrency
  model before introducing concurrent mutation. Define duplicate/stale command behavior
  in the relevant spec rather than inventing it while implementing.
- Return useful errors at public boundaries. Use context for cancellable I/O and
  resource lifetimes; propagate failure rather than log-and-continue by default.
- Test behavior through agreed public interfaces. Expected results come from the
  spec or independently worked examples, not from recomputing the implementation.
- Avoid sleeps and uncontrolled randomness in logic tests. Test meaningful invariants
  and failure behavior when those are part of the agreed scope.
- Reviews must assess both compliance with these standards and fidelity to the spec.
  Automated formatting, vet, tests, and build are separate evidence.
