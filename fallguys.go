package main

// Fall Guys specifics on top of the EOS core.
//
// Fall Guys (Mediatonic / Epic) runs on EOS: a console gets an EOS client token,
// connects its identity to a Product User Id, then joins "show" lobbies through
// matchmaking. This branch preps the Fall-Guys-specific values so the core serves
// it; the exact EOS ids are Epic's and must be read from a real client/capture
// (see NOTES.md), so they default to placeholders here.

func init() {
	// The core reads EOS_DEPLOYMENT_ID from the environment; default it to the
	// Fall Guys deployment on this branch when the operator has not set one.
	if deployment == "" {
		deployment = envOr("EOS_DEPLOYMENT_ID", fallGuysDeployment)
	}
}

const (
	// UNCONFIRMED: the real Fall Guys EOS deployment/sandbox/client ids come from
	// the game's EOS config or a capture. Placeholders until then.
	fallGuysDeployment = "fallguys-deployment-TO-CONFIRM"

	// Fall Guys groups matchmaking by show; lobbies use this bucket prefix.
	fallGuysBucketPrefix = "fallguys:show"
)
