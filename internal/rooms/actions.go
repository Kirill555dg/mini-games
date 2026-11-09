package rooms

// Action names are shared with the generated public API contract.
const (
	ActionCreate     = "create"
	ActionEnter      = "enter"
	ActionLeave      = "leave"
	ActionReady      = "ready"
	ActionSeat       = "seat"
	ActionAcceptSwap = "accept-swap"
	ActionCancelSwap = "cancel-swap"
	ActionMove       = "move"
)

func Actions() map[string]string {
	return map[string]string{"create": ActionCreate, "enter": ActionEnter, "leave": ActionLeave, "ready": ActionReady, "seat": ActionSeat, "acceptSwap": ActionAcceptSwap, "cancelSwap": ActionCancelSwap, "move": ActionMove}
}
