package types

// BitStatus .
type BitStatus uint

// Match .
func (status BitStatus) Match(flags ...BitStatus) bool {
	for _, flag := range flags {
		if status&flag == 0 {
			return false
		}
	}
	return true
}

// Mark .
func (status *BitStatus) Mark(flags ...BitStatus) {
	s := uint(*status)
	for _, flag := range flags {
		s |= uint(flag)
	}
	*status = BitStatus(s)
}

// Unmark .
func (status *BitStatus) Unmark(flags ...BitStatus) {
	s := uint(*status)
	for _, flag := range flags {
		s &= (^uint(flag))
	}
	*status = BitStatus(s)
}
