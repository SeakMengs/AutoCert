package repository

type FileRepository struct {
	*baseRepository
}

// func (fr FileRepository) Create(ctx context.Context, tx *gorm.DB, file *model.File) (*model.File, error) {
// 	fr.logger.Debugf("Create file with data: %v \n", file)

// 	db := fr.getDB(tx)
// 	ctx, cancel := context.WithTimeout(ctx, constant.QUERY_TIMEOUT_DURATION)
// 	defer cancel()

// 	if err := db.WithContext(ctx).Model(&model.File{}).Create(file).Error; err != nil {
// 		return file, err
// 	}

// 	return file, nil
// }

// func (fr FileRepository) Delete(ctx context.Context, tx *gorm.DB, fileID string) error {
// 	fr.logger.Debugf("Delete file with fileID: %s \n", fileID)

// 	db := fr.getDB(tx)
// 	ctx, cancel := context.WithTimeout(ctx, constant.QUERY_TIMEOUT_DURATION)
// 	defer cancel()

// 	if err := db.WithContext(ctx).Model(&model.File{}).Where(&model.File{
// 		BaseModel: model.BaseModel{
// 			ID: fileID,
// 		},
// 	}).Delete(&model.File{}).Error; err != nil {
// 		return err
// 	}

// 	return nil
// }

// func (fr FileRepository) Update(ctx context.Context, tx *gorm.DB, file *model.File) (*model.File, error) {
// 	fr.logger.Debugf("Update file with data: %v \n", file)

// 	db := fr.getDB(tx)
// 	ctx, cancel := context.WithTimeout(ctx, constant.QUERY_TIMEOUT_DURATION)
// 	defer cancel()

// 	if err := db.WithContext(ctx).Model(&model.File{}).Where(&model.File{
// 		BaseModel: model.BaseModel{
// 			ID: file.ID,
// 		},
// 	}).Updates(file).Error; err != nil {
// 		return file, err
// 	}

// 	return file, nil
// }
