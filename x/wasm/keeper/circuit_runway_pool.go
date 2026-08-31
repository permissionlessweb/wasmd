package keeper

import (
	"context"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/CosmWasm/wasmd/x/wasm/types"
)

func (k Keeper) receiveCircuitRunwayPayment(ctx context.Context, payer sdk.AccAddress, amt sdk.Coins) error {
	valPool, devPool := types.SplitCircuitRunway(amt)
	if !valPool.Empty() {
		if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, payer, types.CircuitValPoolName, valPool); err != nil {
			return err
		}
	}
	if !devPool.Empty() {
		if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, payer, types.CircuitDevPoolName, devPool); err != nil {
			return err
		}
	}
	return nil
}

// DistributeCircuitValPool pays the operator-equal share of circuit rent.
// Remainder (amount % n) stays in the pool. No-op if empty or no bonded vals.
func (k Keeper) DistributeCircuitValPool(ctx context.Context) {
	if k.stakingKeeper == nil {
		return
	}
	vals, err := k.stakingKeeper.GetBondedValidatorsByPower(ctx)
	if err != nil || len(vals) == 0 {
		return
	}
	poolAddr := authtypes.NewModuleAddress(types.CircuitValPoolName)
	bal := k.bankKeeper.GetAllBalances(ctx, poolAddr)
	if bal.Empty() {
		return
	}
	n := int64(len(vals))
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	paid := sdk.NewCoins()
	for _, c := range bal {
		each := c.Amount.QuoRaw(n)
		if !each.IsPositive() {
			continue
		}
		paid = paid.Add(sdk.NewCoin(c.Denom, each.MulRaw(n)))
		share := sdk.NewCoins(sdk.NewCoin(c.Denom, each))
		for _, v := range vals {
			op, err := sdk.ValAddressFromBech32(v.OperatorAddress)
			if err != nil {
				continue
			}
			if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.CircuitValPoolName, sdk.AccAddress(op), share); err != nil {
				sdkCtx.Logger().Debug("circuit val pool payout", "err", err)
			}
		}
	}
	if !paid.Empty() {
		sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
			"circuit_val_pool_payout",
			sdk.NewAttribute("operators", strconv.FormatInt(n, 10)),
			sdk.NewAttribute("paid", paid.String()),
		))
	}
}
