package database

func (d *Database) GetUserByID(id int64) (*User, error) {
	var user User
	err := d.db.
		Where(&User{
			ID: id,
		}).
		First(&user).Error
	return &user, err
}

func (d *Database) CreateUser(user *User) error {
	return d.db.Create(user).Error
}

func (d *Database) GetAllUserIDs() ([]int64, error) {
	var ids []int64
	err := d.db.
		Model(&User{}).
		Pluck("id", &ids).Error

	return ids, err
}
