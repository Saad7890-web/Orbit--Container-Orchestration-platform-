package daemon

import (
	"context"
	"fmt"
	"time"
)

func (d *Daemon) Shutdown(ctx context.Context) error {
	if d == nil {
		return nil
	}

	shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if d.httpServer != nil {
		if err := d.httpServer.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown http server: %w", err)
		}
	}

	if d.triggerEng != nil {
		_ = d.triggerEng.Sync("", "", nil)
	}

	if d.runtime != nil {
		_ = d.runtime.Close()
	}
	if d.store != nil {
		_ = d.store.Close()
	}
	if d.bus != nil {
		_ = d.bus.Close()
	}

	return nil
}