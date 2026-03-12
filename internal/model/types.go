package model

// Entry represents a single item in the Vault hierarchy.
type Entry struct {
	Name  string
	IsDir bool
}

// Secret holds key-value pairs for a Vault secret.
type Secret struct {
	Path string
	Data map[string]string
	Keys []string // ordered keys for stable display
}

// PreviewMode controls how the right-pane preview renders secrets.
type PreviewMode int

const (
	PreviewHidden PreviewMode = iota // show keys with ********
	PreviewValues                    // show actual values
	PreviewJSON                      // show as JSON
)

// ViewMode represents the current UI mode.
type ViewMode int

const (
	ModeExplorer ViewMode = iota
	ModeSecret
	ModeSecretEdit // inline editing a value in secret popup
	ModeConfirm
	ModeInput
	ModeSearch         // jump-to search (s)
	ModeFilter         // filter entries (/)
	ModeVersionHistory // secret version history view
	ModeHelp           // full-screen help overlay
	ModeJumpPath       // jump-to-path prompt (S)
	ModeBookmark       // bookmark overlay
	ModeThemePicker    // colorscheme picker overlay
)

// UndoType represents the kind of action that can be undone.
type UndoType int

const (
	UndoCreateSecret UndoType = iota // undo = delete the created secret
	UndoDeleteSecret                 // undo = recreate with stored data
	UndoCutSecret                    // undo = recreate at original path (legacy, unused)
	UndoRenameSecret                 // undo = rename back (Path=new, OldPath=old)
	UndoPasteSecret                  // undo = delete pasted secret
	UndoCutPaste                     // undo = delete dest (Path) + recreate source (OldPath)
	UndoEditSecret                   // undo = restore full secret snapshot
)

// UndoAction stores enough info to reverse an operation.
type UndoAction struct {
	Type        UndoType
	Description string
	Path        string            // affected secret path
	Data        map[string]string // secret data snapshot (for restore)
	Keys        []string          // key order snapshot (for restore)
	OldPath     string            // for rename: the original path before rename
}

// SecretVersion holds version metadata for a Vault KV v2 secret.
type SecretVersion struct {
	Version      string
	CreatedTime  string
	DeletionTime string
	Destroyed    bool
}

// InputAction represents what the input prompt is for.
type InputAction int

const (
	InputNone InputAction = iota
	InputRename
	InputNewSecret
	InputNewSecretEditor // new secret via external editor
	InputNewKey
	InputNewValue
	InputEditValue
)
