package sockets

type Activator struct {
	UnitPath    string
	ServicePath string
}

func NewActivator(unitPath, servicePath string) *Activator {
	return &Activator{
		UnitPath:    unitPath,
		ServicePath: servicePath,
	}
}

func (a *Activator) Run() error {
	return nil
}
