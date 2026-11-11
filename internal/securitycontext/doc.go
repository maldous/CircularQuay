// Package securitycontext verifies the pure, signed v1 security-context boundary.
//
// The package intentionally has no configuration, CLI, provider, state-store,
// network, or service dependency. Its caller must supply an already protected
// enrollment binding, trust record, aggregate accepted-state snapshot, and
// trusted time. Capability claims remain declared requirements; protected
// compiler and admission code own atomic runtime measurement and enforcement. This
// separation prevents a policy,
// SDK, request, or browser control from selecting its own trust or profile.
package securitycontext
