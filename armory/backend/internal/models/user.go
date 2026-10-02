package models

type User struct {
	ID        int64
	Name      string
	ServiceNo string
	Role      string
	Faces     int64
}

type FaceEnrollmentRequest struct {
	ID        int64
	Name      string
	ServiceNo string
	FaceRefs  string
	Status    string
	CreatedAt string
}

type FaceEnrollment struct {
	User    User
	FaceRef string
}
