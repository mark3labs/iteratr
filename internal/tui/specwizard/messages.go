package specwizard

// TitleSubmittedMsg is sent when the user submits the title.
type TitleSubmittedMsg struct {
	Title string
}

// DescriptionSubmittedMsg is sent when the user submits the description.
type DescriptionSubmittedMsg struct {
	Description string
}

// SpecContentReceivedMsg is sent when the agent finishes generating the spec.
type SpecContentReceivedMsg struct {
	Content string
}

// SpecSavedMsg is sent when the spec has been saved to disk.
type SpecSavedMsg struct {
	Path string
}
