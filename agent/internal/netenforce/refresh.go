package netenforce

import "context"

func refreshOnce(ctx context.Context, enf Enforcer, spec Spec, serverURL, lastSig string) (string, error) {
	spec = EnsureControlPlaneSafety(spec, serverURL)
	expanded := expandHosts(spec)
	sig := specSignature(expanded)
	if sig == lastSig {
		return lastSig, nil
	}
	if err := enf.Apply(ctx, expanded); err != nil {
		return lastSig, err
	}
	return sig, nil
}

func RefreshHosts(ctx context.Context, spec Spec, serverURL, lastSig string) (string, error) {
	return refreshOnce(ctx, New(), spec, serverURL, lastSig)
}