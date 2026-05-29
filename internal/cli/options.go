package cli

type PlanOptions struct {
	Inputs []string
	Config string
}

type ApplyOptions struct {
	Inputs      []string
	AutoApprove bool
	DeleteMode  string
}
