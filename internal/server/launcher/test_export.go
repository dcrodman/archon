package launcher

import "net"

func (l *Launcher) GetFrontends() []*frontend {
	return l.frontends
}

func (f *frontend) Name() string {
	return f.backend.Name()
}

func (f *frontend) Addr() net.Addr {
	return f.listenAddr
}
