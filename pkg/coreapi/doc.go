// Package coreapi is a client for the FABRIC Core API (the "UIS" service at
// https://uis.fabric-testbed.net). It resolves the caller's identity and
// bastion login, complementing pkg/client (the orchestrator API): the
// orchestrator owns slices and slivers, while the Core API owns people,
// projects, and SSH keys.
//
// Tooling that SSHes through the FABRIC bastion resolves the caller's
// bastion_login here when it is not already configured, then dials the bastion
// itself — the Core API performs no SSH. Nothing on the orchestrator path
// depends on this package.
package coreapi
