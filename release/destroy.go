package release

import (
	"context"

	"github.com/hashicorp/waypoint-plugin-sdk/terminal"
)

// Implement the Destroyer interface
func (rm *ReleaseManager) DestroyFunc() interface{} {
	return rm.destroy
}

func (rm *ReleaseManager) destroy(ctx context.Context, ui terminal.UI, release *Release) error {
	return nil
}
