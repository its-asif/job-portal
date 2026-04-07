package models

// RegisterRequest is the request body for registering a user.
type RegisterRequest struct {
	Name     string `json:"name" example:"Asif"`
	Email    string `json:"email" example:"asif@example.com"`
	Password string `json:"password" example:"secret123"`
	Role     string `json:"role" example:"jobseeker"`
}

// LoginRequest is the request body for user login.
type LoginRequest struct {
	Email    string `json:"email" example:"asif@example.com"`
	Password string `json:"password" example:"secret123"`
}

// CreateJobRequest is the request body for creating a job.
type CreateJobRequest struct {
	Title       string `json:"title" example:"Backend Engineer"`
	Description string `json:"description" example:"Build scalable APIs"`
	Location    string `json:"location" example:"Dhaka"`
	Salary      int64  `json:"salary" example:"90000"`
	Company     string `json:"company" example:"TalentDock"`
}

// UpdateJobRequest is the request body for updating a job.
type UpdateJobRequest struct {
	Title       *string `json:"title,omitempty" example:"Senior Backend Engineer"`
	Description *string `json:"description,omitempty" example:"Build and maintain APIs"`
	Location    *string `json:"location,omitempty" example:"Remote"`
	Salary      *int64  `json:"salary,omitempty" example:"120000"`
	Company     *string `json:"company,omitempty" example:"TalentDock"`
}

// UpdateApplicationStatusRequest is the request body for changing application status.
type UpdateApplicationStatusRequest struct {
	Status string `json:"status" example:"reviewed"`
}
