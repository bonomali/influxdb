package label

import (
	"context"
	"errors"
	"fmt"

	"github.com/influxdata/influxdb/v2"
	"github.com/influxdata/influxdb/v2/authorizer"
	icontext "github.com/influxdata/influxdb/v2/context"
	kithttp "github.com/influxdata/influxdb/v2/kit/transport/http"
)

var _ influxdb.LabelService = (*AuthedLabelService)(nil)

type AuthedLabelService struct {
	s influxdb.LabelService
}

// NewAuthedLabelService constructs an instance of an authorizing label serivce.
func NewAuthedLabelService(s influxdb.LabelService) *AuthedLabelService {
	return &AuthedLabelService{
		s: s,
	}
}
func (s *AuthedLabelService) CreateLabel(ctx context.Context, l *influxdb.Label) error {
	if _, _, err := authorizer.AuthorizeCreate(ctx, influxdb.LabelsResourceType, l.OrgID); err != nil {
		return err
	}
	return s.s.CreateLabel(ctx, l)
}

func (s *AuthedLabelService) FindLabels(ctx context.Context, filter influxdb.LabelFilter, opt ...influxdb.FindOptions) ([]*influxdb.Label, error) {
	// TODO: we'll likely want to push this operation into the database eventually since fetching the whole list of data
	// will likely be expensive.
	ls, err := s.s.FindLabels(ctx, filter, opt...)
	if err != nil {
		return nil, err
	}
	ls, _, err = authorizer.AuthorizeFindLabels(ctx, ls)
	return ls, err
}

// FindLabelByID checks to see if the authorizer on context has read access to the label id provided.
func (s *AuthedLabelService) FindLabelByID(ctx context.Context, id influxdb.ID) (*influxdb.Label, error) {
	l, err := s.s.FindLabelByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if _, _, err := authorizer.AuthorizeRead(ctx, influxdb.LabelsResourceType, id, l.OrgID); err != nil {
		return nil, err
	}
	return l, nil
}

// FindResourceLabels retrieves all labels belonging to the filtering resource if the authorizer on context has read access to it.
// Then it filters the list down to only the labels that are authorized.
func (s *AuthedLabelService) FindResourceLabels(ctx context.Context, filter influxdb.LabelMappingFilter) ([]*influxdb.Label, error) {
	if err := filter.ResourceType.Valid(); err != nil {
		return nil, err
	}
	// first fetch all labels for this resource
	ls, err := s.s.FindResourceLabels(ctx, filter)
	if err != nil {
		return nil, err
	}

	// check the permissions for the resource by the org on the context
	orgID := kithttp.OrgIDFromContext(ctx)
	if orgID == nil {
		return nil, errors.New("failed to find orgID on context")
	}

	if _, _, err := authorizer.AuthorizeRead(ctx, filter.ResourceType, filter.ResourceID, *orgID); err != nil {
		return nil, err
	}

	// then filter the labels we got to return only the ones the user is authorized to read
	ls, _, err = authorizer.AuthorizeFindLabels(ctx, ls)
	return ls, err
}

// UpdateLabel checks to see if the authorizer on context has write access to the label provided.
func (s *AuthedLabelService) UpdateLabel(ctx context.Context, id influxdb.ID, upd influxdb.LabelUpdate) (*influxdb.Label, error) {
	l, err := s.s.FindLabelByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if _, _, err := authorizer.AuthorizeWrite(ctx, influxdb.LabelsResourceType, l.ID, l.OrgID); err != nil {
		return nil, err
	}
	return s.s.UpdateLabel(ctx, id, upd)
}

// DeleteLabel checks to see if the authorizer on context has write access to the label provided.
func (s *AuthedLabelService) DeleteLabel(ctx context.Context, id influxdb.ID) error {
	l, err := s.s.FindLabelByID(ctx, id)
	if err != nil {
		return err
	}
	if _, _, err := authorizer.AuthorizeWrite(ctx, influxdb.LabelsResourceType, l.ID, l.OrgID); err != nil {
		return err
	}
	return s.s.DeleteLabel(ctx, id)
}

// CreateLabelMapping checks to see if the authorizer on context has write access to the label and the resource contained by the label mapping in creation.
func (s *AuthedLabelService) CreateLabelMapping(ctx context.Context, m *influxdb.LabelMapping) error {
	fmt.Println("authorizing create label mapping ")
	fmt.Println("resource ID ", m.ResourceID)
	fmt.Println("label ID ", m.LabelID)
	l, err := s.s.FindLabelByID(ctx, m.LabelID)
	if err != nil {
		return err
	}
	auth, _ := icontext.GetAuthorizer(ctx)
	fmt.Println(auth.PermissionSet())
	fmt.Println(kithttp.OrgIDFromContext(ctx))
	if _, _, err := authorizer.AuthorizeWrite(ctx, influxdb.LabelsResourceType, m.LabelID, l.OrgID); err != nil {
		return err
	}
	if _, _, err := authorizer.AuthorizeWrite(ctx, m.ResourceType, m.ResourceID, l.OrgID); err != nil {
		return err
	}
	return s.s.CreateLabelMapping(ctx, m)
}

// DeleteLabelMapping checks to see if the authorizer on context has write access to the label and the resource of the label mapping to delete.
func (s *AuthedLabelService) DeleteLabelMapping(ctx context.Context, m *influxdb.LabelMapping) error {
	l, err := s.s.FindLabelByID(ctx, m.LabelID)
	if err != nil {
		return err
	}
	if _, _, err := authorizer.AuthorizeWrite(ctx, influxdb.LabelsResourceType, m.LabelID, l.OrgID); err != nil {
		return err
	}
	if _, _, err := authorizer.AuthorizeWrite(ctx, m.ResourceType, m.ResourceID, l.OrgID); err != nil {
		return err
	}
	return s.s.DeleteLabelMapping(ctx, m)
}
