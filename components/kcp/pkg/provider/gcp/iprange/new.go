package iprange

import (
	"context"
	"fmt"

	"github.com/kyma-project/cloud-manager/components/kcp/pkg/common/actions/focal"
	"github.com/kyma-project/cloud-manager/components/kcp/pkg/iprange/types"
	"github.com/kyma-project/cloud-manager/components/lib/composed"
)

func New(stateFactory StateFactory) composed.Action {
	return func(ctx context.Context, st composed.State) (error, context.Context) {

		logger := composed.LoggerFromCtx(ctx)
		state, err := stateFactory.NewState(ctx, st.(types.State))
		if err != nil {
			err = fmt.Errorf("error creating new gcp iprange state: %w", err)
			logger.Error(err, "Error")
			return composed.StopAndForget, nil
		}
		return composed.ComposeActions(
			"gcpIpRange",
			validateCidr,
			focal.AddFinalizer,
			loadAddress,
			loadPsaConnection,
			compareStates,
			composed.BuildSwitchAction(
				"gcpIpRangeSwitch",
				composed.ComposeActions("SyncGCP", syncAddress, syncPsaConnection),
				composed.NewCase(DeletePending, composed.ComposeActions("DeleteIPRange", syncPsaConnection, syncAddress)),
				composed.NewCase(Deleted, focal.RemoveFinalizer),
				composed.NewCase(InSync, switchToReadyState),
			),
		)(ctx, state)
	}
}

func Deleted(_ context.Context, st composed.State) bool {
	state := st.(*State)
	return state.inSync && !state.Obj().GetDeletionTimestamp().IsZero()
}

func DeletePending(_ context.Context, st composed.State) bool {
	state := st.(*State)
	return !state.inSync && !state.Obj().GetDeletionTimestamp().IsZero()
}

func InSync(_ context.Context, st composed.State) bool {
	state := st.(*State)
	return state.inSync && state.Obj().GetDeletionTimestamp().IsZero()
}
