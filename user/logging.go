package user

import (
	"golang.org/x/net/context"

	"github.com/weaveworks/common/logging"
)

// LogWith returns user and org information from the context as log fields.
func LogWith(ctx context.Context, log logging.Interface) logging.Interface {
	userID, err := ExtractUserID(ctx)
	if err == nil {
		log = log.WithField("userID", userID)
	}

	orgIDs, err := ExtractOrgIDs(ctx)
	if err == nil {
		if len(orgIDs) == 1 {
			log = log.WithField("orgID", orgIDs[0])
		} else {
			log = log.WithField("orgIDs", orgIDs)
		}
	}

	return log
}
