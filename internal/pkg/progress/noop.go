package progress

// NoopProgress — пустая реализация Progress, которая ничего не делает.
// Используется когда progress отключён (BR_SHOW_PROGRESS=false).
type NoopProgress struct{}

// NewNoOp создаёт NoopProgress — пустую реализацию Progress.
func NewNoOp() Progress {
	return &NoopProgress{}
}

// Start ничего не делает.
func (p *NoopProgress) Start(_ string) {}

// Update ничего не делает.
func (p *NoopProgress) Update(_ int64, _ string) {}

// SetTotal ничего не делает.
func (p *NoopProgress) SetTotal(_ int64) {}

// Finish ничего не делает.
func (p *NoopProgress) Finish() {}
