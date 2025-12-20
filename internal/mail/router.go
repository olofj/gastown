package mail

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/steveyegge/gastown/internal/tmux"
)

// Router handles message delivery via beads.
type Router struct {
	workDir string // directory to run bd commands in
	tmux    *tmux.Tmux
}

// NewRouter creates a new mail router.
// workDir should be a directory containing a .beads database.
func NewRouter(workDir string) *Router {
	return &Router{
		workDir: workDir,
		tmux:    tmux.NewTmux(),
	}
}

// Send delivers a message via beads message.
func (r *Router) Send(msg *Message) error {
	// Convert addresses to beads identities
	toIdentity := addressToIdentity(msg.To)
	fromIdentity := addressToIdentity(msg.From)

	// Build command: bd mail send <recipient> -s <subject> -m <body>
	args := []string{"mail", "send", toIdentity,
		"-s", msg.Subject,
		"-m", msg.Body,
	}

	// Add priority flag
	beadsPriority := PriorityToBeads(msg.Priority)
	args = append(args, "--priority", fmt.Sprintf("%d", beadsPriority))

	// Add message type if set
	if msg.Type != "" && msg.Type != TypeNotification {
		args = append(args, "--type", string(msg.Type))
	}

	// Add thread ID if set
	if msg.ThreadID != "" {
		args = append(args, "--thread-id", msg.ThreadID)
	}

	// Add reply-to if set
	if msg.ReplyTo != "" {
		args = append(args, "--reply-to", msg.ReplyTo)
	}

	cmd := exec.Command("bd", args...)
	cmd.Env = append(cmd.Environ(), "BEADS_AGENT_NAME="+fromIdentity)
	cmd.Dir = r.workDir

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg != "" {
			return errors.New(errMsg)
		}
		return fmt.Errorf("sending message: %w", err)
	}

	// Handle delivery based on mode
	if msg.Delivery == DeliveryInterrupt {
		// Interrupt: inject system-reminder directly into session
		r.interruptRecipient(msg)
	} else {
		// Queue (default): just notify in status line
		r.notifyRecipient(msg)
	}

	return nil
}

// GetMailbox returns a Mailbox for the given address.
func (r *Router) GetMailbox(address string) (*Mailbox, error) {
	return NewMailboxFromAddress(address, r.workDir), nil
}

// notifyRecipient sends a notification to a recipient's tmux session.
// Uses display-message for non-disruptive notification.
// Supports mayor/, rig/polecat, and rig/refinery addresses.
func (r *Router) notifyRecipient(msg *Message) error {
	sessionID := addressToSessionID(msg.To)
	if sessionID == "" {
		return nil // Unable to determine session ID
	}

	// Check if session exists
	hasSession, err := r.tmux.HasSession(sessionID)
	if err != nil || !hasSession {
		return nil // No active session, skip notification
	}

	// Display notification in status line (non-disruptive)
	notification := fmt.Sprintf("[MAIL] From %s: %s", msg.From, msg.Subject)
	return r.tmux.DisplayMessageDefault(sessionID, notification)
}

// interruptRecipient injects a system-reminder directly into the session.
// Uses tmux send-keys to inject text that Claude will see as input.
// This is disruptive - use for lifecycle events, URGENT messages, or stuck detection.
func (r *Router) interruptRecipient(msg *Message) error {
	sessionID := addressToSessionID(msg.To)
	if sessionID == "" {
		return nil // Unable to determine session ID
	}

	// Check if session exists
	hasSession, err := r.tmux.HasSession(sessionID)
	if err != nil || !hasSession {
		return nil // No active session, skip interrupt
	}

	// Build system-reminder with message content
	priorityStr := ""
	if msg.Priority == PriorityUrgent {
		priorityStr = " [URGENT]"
	} else if msg.Priority == PriorityHigh {
		priorityStr = " [HIGH PRIORITY]"
	}

	reminder := fmt.Sprintf("\n<system-reminder>\n📬 NEW MAIL%s from %s\nSubject: %s\n", priorityStr, msg.From, msg.Subject)
	if msg.Body != "" {
		reminder += fmt.Sprintf("\n%s\n", msg.Body)
	}
	reminder += "\nRun 'gt mail inbox' to see your messages.\n</system-reminder>\n"

	// Inject via send-keys (don't press Enter, just paste)
	return r.tmux.SendKeysRaw(sessionID, reminder)
}

// addressToSessionID converts a mail address to a tmux session ID.
// Returns empty string if address format is not recognized.
func addressToSessionID(address string) string {
	// Mayor address: "mayor/" or "mayor"
	if strings.HasPrefix(address, "mayor") {
		return "gt-mayor"
	}

	// Rig-based address: "rig/target"
	parts := strings.SplitN(address, "/", 2)
	if len(parts) != 2 || parts[1] == "" {
		return ""
	}

	rig := parts[0]
	target := parts[1]

	// Polecat: gt-rig-polecat
	// Refinery: gt-rig-refinery (if refinery has its own session)
	return fmt.Sprintf("gt-%s-%s", rig, target)
}
