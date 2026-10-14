package decision

import "context"

// Service runs structured decisions through a configured backend.
type Service struct {
	Backend  Backend
	DriverID string
	BackendName string
}

// Decide executes a decision request.
func (s *Service) Decide(ctx context.Context, req Request) (*Response, error) {
	if s == nil || s.Backend == nil {
		return nil, ErrDisabled
	}
	resp, err := s.Backend.Decide(ctx, req)
	if err != nil {
		return nil, err
	}
	if resp != nil {
		if resp.Driver == "" {
			resp.Driver = s.DriverID
		}
		if resp.Backend == "" {
			resp.Backend = s.BackendName
		}
	}
	return resp, nil
}

// DescribeBackend returns a short label for logging.
func (s *Service) DescribeBackend() string {
	if s == nil {
		return ""
	}
	if s.DriverID != "" {
		return s.BackendName + "/" + s.DriverID
	}
	return s.BackendName
}

// Close releases backend resources when supported.
func (s *Service) Close() error {
	if s == nil {
		return nil
	}
	if c, ok := s.Backend.(interface{ Close() error }); ok {
		return c.Close()
	}
	return nil
}
