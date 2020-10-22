package user

import (
	"reflect"

	"golang.org/x/net/context"
	"google.golang.org/grpc/metadata"
)

// ExtractFromGRPCRequest extracts the user ID from the request metadata and returns
// the user ID and a context with the user ID injected.
func ExtractFromGRPCRequest(ctx context.Context) (string, context.Context, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", ctx, ErrNoOrgID
	}

	orgIDs, ok := md[lowerOrgIDHeaderName]
	if !ok || len(orgIDs) < 1 {
		return "", ctx, ErrNoOrgID
	}

	return orgIDs[0], InjectOrgIDs(ctx, orgIDs), nil
}

// InjectIntoGRPCRequest injects the orgIDs from the context into the request metadata.
func InjectIntoGRPCRequest(ctx context.Context) (context.Context, error) {
	orgIDs, err := ExtractOrgIDs(ctx)
	if err != nil {
		return ctx, err
	}

	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		md = metadata.New(map[string]string{})
	}
	newCtx := ctx
	if existingIDs, ok := md[lowerOrgIDHeaderName]; ok {
		if !reflect.DeepEqual(orgIDs, existingIDs) {
			return ctx, ErrTooManyOrgIDs
		}
	} else {
		md = md.Copy()
		md[lowerOrgIDHeaderName] = orgIDs
		newCtx = metadata.NewOutgoingContext(ctx, md)
	}
	return newCtx, nil
}
