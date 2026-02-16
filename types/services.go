package types

type ServiceTypeT string

const (
	BalanceGuardService ServiceTypeT = "balance_guard"
	MaxGuardService     ServiceTypeT = "max_guard"
	PriceGuardService   ServiceTypeT = "price_guard"
	EnergyGuardService  ServiceTypeT = "energy_guard"
	ScheduleService     ServiceTypeT = "schedule"
)
