package product

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	dberrors "github.com/zone3-labs/mancing-id/internal/errors"
	productDb "github.com/zone3-labs/mancing-id/internal/product/db"
)

type productPostgresRepository struct {
	db      *sql.DB
	queries *productDb.Queries
}

func NewProductRepository(db *sql.DB) ProductRepository {
	return &productPostgresRepository{db: db, queries: productDb.New(db)}
}

func (p productPostgresRepository) CreateProduct(ctx context.Context, product *Product) error {
	return p.queries.CreateProduct(ctx, productDb.CreateProductParams{
		ID:               product.ID,
		Name:             product.Name,
		Slug:             product.Slug,
		Description:      product.Description,
		ShortDescription: product.ShortDescription,
		Status:           productDb.ProductStatus(product.Status),
		BrandID:          product.BrandID,
		IsFeatured:       product.IsFeature,
	})
}

func (p productPostgresRepository) GetAllProducts(ctx context.Context) ([]*Product, error) {
	rows, err := p.queries.GetAllProducts(ctx)
	if err != nil {
		return nil, err
	}
	products := make([]*Product, 0, len(rows))
	for _, row := range rows {
		products = append(products, &Product{
			ID:               row.ID,
			Name:             row.Name,
			Slug:             row.Slug,
			Description:      row.Description,
			ShortDescription: row.ShortDescription,
			BrandID:          row.BrandID,
			Status:           ProductStatus(row.Status),
			IsFeature:        row.IsFeatured,
			CreatedAt:        row.CreatedAt,
			UpdatedAt:        row.UpdatedAt,
			DeletedAt:        row.DeletedAt,
		})
	}
	return products, nil
}

func (p productPostgresRepository) GetProductBySlug(ctx context.Context, slug string) (*ProductDetail, error) {
	rows, err := p.queries.GetProductDetailBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, ErrProductNotFound
	}

	first := rows[0]
	detail := &ProductDetail{
		Product: Product{
			ID:               first.ID,
			Name:             first.Name,
			Slug:             first.Slug,
			Description:      first.Description,
			ShortDescription: first.ShortDescription,
			BrandID:          first.BrandID,
			Status:           ProductStatus(first.Status),
			IsFeature:        first.IsFeatured,
			CreatedAt:        first.CreatedAt,
			UpdatedAt:        first.UpdatedAt,
			DeletedAt:        first.DeletedAt,
		},
		Options: make([]*ProductOptionWithValues, 0),
	}

	// Track option insertion order and map for dedup
	optionIndex := make(map[uuid.UUID]int)

	for _, row := range rows {
		if !row.OptionID.Valid {
			continue // product has no options
		}
		optID := row.OptionID.UUID

		if _, seen := optionIndex[optID]; !seen {
			opt := &ProductOptionWithValues{
				ProductOption: ProductOption{
					ID:        optID,
					ProductID: first.ID,
					Name:      row.OptionName.String,
					Position:  row.OptionPosition.Int16,
					IsActive:  row.OptionIsActive.Bool,
					CreatedAt: row.OptionCreatedAt.Time,
					UpdatedAt: row.OptionUpdatedAt.Time,
				},
				Values: make([]*ProductOptionValue, 0),
			}
			optionIndex[optID] = len(detail.Options)
			detail.Options = append(detail.Options, opt)
		}

		if !row.ValueID.Valid {
			continue // option has no values yet
		}
		detail.Options[optionIndex[optID]].Values = append(
			detail.Options[optionIndex[optID]].Values,
			&ProductOptionValue{
				ID:              row.ValueID.UUID,
				ProductOptionID: optID,
				Value:           row.ValueText.String,
				Position:        row.ValuePosition.Int16,
				IsActive:        row.ValueIsActive.Bool,
				CreatedAt:       row.ValueCreatedAt.Time,
				UpdatedAt:       row.ValueUpdatedAt.Time,
			},
		)
	}

	imgRows, err := p.queries.GetProductImagesByProductID(ctx, first.ID)
	if err != nil {
		return nil, err
	}
	detail.Images = make([]*ProductImage, 0, len(imgRows))
	for _, imgRow := range imgRows {
		detail.Images = append(detail.Images, mapProductImage(imgRow))
	}

	variants, err := p.GetProductVariants(ctx, first.ID)
	if err != nil {
		return nil, err
	}
	detail.Variants = variants

	return detail, nil
}

func (p productPostgresRepository) GetProductDetailByID(ctx context.Context, id uuid.UUID) (*ProductDetail, error) {
	rows, err := p.queries.GetProductDetailByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, ErrProductNotFound
	}

	first := rows[0]
	detail := &ProductDetail{
		Product: Product{
			ID:               first.ID,
			Name:             first.Name,
			Slug:             first.Slug,
			Description:      first.Description,
			ShortDescription: first.ShortDescription,
			BrandID:          first.BrandID,
			Status:           ProductStatus(first.Status),
			IsFeature:        first.IsFeatured,
			CreatedAt:        first.CreatedAt,
			UpdatedAt:        first.UpdatedAt,
			DeletedAt:        first.DeletedAt,
		},
		Options: make([]*ProductOptionWithValues, 0),
	}

	// Track option insertion order and map for dedup
	optionIndex := make(map[uuid.UUID]int)

	for _, row := range rows {
		if !row.OptionID.Valid {
			continue // product has no options
		}
		optID := row.OptionID.UUID

		if _, seen := optionIndex[optID]; !seen {
			opt := &ProductOptionWithValues{
				ProductOption: ProductOption{
					ID:        optID,
					ProductID: first.ID,
					Name:      row.OptionName.String,
					Position:  row.OptionPosition.Int16,
					IsActive:  row.OptionIsActive.Bool,
					CreatedAt: row.OptionCreatedAt.Time,
					UpdatedAt: row.OptionUpdatedAt.Time,
				},
				Values: make([]*ProductOptionValue, 0),
			}
			optionIndex[optID] = len(detail.Options)
			detail.Options = append(detail.Options, opt)
		}

		if !row.ValueID.Valid {
			continue // option has no values yet
		}
		detail.Options[optionIndex[optID]].Values = append(
			detail.Options[optionIndex[optID]].Values,
			&ProductOptionValue{
				ID:              row.ValueID.UUID,
				ProductOptionID: optID,
				Value:           row.ValueText.String,
				Position:        row.ValuePosition.Int16,
				IsActive:        row.ValueIsActive.Bool,
				CreatedAt:       row.ValueCreatedAt.Time,
				UpdatedAt:       row.ValueUpdatedAt.Time,
			},
		)
	}

	imgRows, err := p.queries.GetProductImagesByProductID(ctx, first.ID)
	if err != nil {
		return nil, err
	}
	detail.Images = make([]*ProductImage, 0, len(imgRows))
	for _, imgRow := range imgRows {
		detail.Images = append(detail.Images, mapProductImage(imgRow))
	}

	variants, err := p.GetProductVariants(ctx, first.ID)
	if err != nil {
		return nil, err
	}
	detail.Variants = variants

	return detail, nil
}

func (p productPostgresRepository) UpdateProduct(ctx context.Context, product *Product) error {
	return p.queries.UpdateProduct(ctx, productDb.UpdateProductParams{
		ID:               product.ID,
		Name:             product.Name,
		Slug:             product.Slug,
		Description:      product.Description,
		ShortDescription: product.ShortDescription,
		Status:           productDb.ProductStatus(product.Status),
		BrandID:          product.BrandID,
		IsFeatured:       product.IsFeature,
	})
}

func (p productPostgresRepository) DeleteProduct(ctx context.Context, id uuid.UUID) error {
	result, err := p.queries.DeleteProduct(ctx, id)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrProductNotFound
	}
	return nil
}

// -------------------------------------------------------
// Options
// -------------------------------------------------------

func (p productPostgresRepository) CreateProductOptionWithValues(ctx context.Context, option *ProductOption, values []*ProductOptionValue) (*ProductOptionWithValues, error) {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	q := productDb.New(tx)

	optRow, err := q.CreateProductOption(ctx, productDb.CreateProductOptionParams{
		ID:        option.ID,
		ProductID: option.ProductID,
		Name:      option.Name,
		Position:  option.Position,
		IsActive:  option.IsActive,
	})
	if dberrors.IsForeignKeyViolation(err) {
		return nil, ErrProductNotFound
	}
	if err != nil {
		return nil, err
	}

	result := &ProductOptionWithValues{
		ProductOption: *mapOption(optRow),
		Values:        make([]*ProductOptionValue, 0, len(values)),
	}

	if len(values) > 0 {
		ids := make([]uuid.UUID, len(values))
		vals := make([]string, len(values))
		positions := make([]int16, len(values))
		actives := make([]bool, len(values))
		for i, v := range values {
			ids[i] = v.ID
			vals[i] = v.Value
			positions[i] = v.Position
			actives[i] = v.IsActive
		}

		valRows, err := q.BulkCreateProductOptionValues(ctx, productDb.BulkCreateProductOptionValuesParams{
			Column1: ids,
			Column2: optRow.ID,
			Column3: vals,
			Column4: positions,
			Column5: actives,
		})
		if err != nil {
			return nil, err
		}
		for _, row := range valRows {
			result.Values = append(result.Values, mapOptionValue(row))
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return result, nil
}

func (p productPostgresRepository) GetProductOptions(ctx context.Context, productID uuid.UUID) ([]*ProductOption, error) {
	rows, err := p.queries.GetProductOptionsByProductID(ctx, productID)
	if err != nil {
		return nil, err
	}
	options := make([]*ProductOption, 0, len(rows))
	for _, row := range rows {
		options = append(options, mapOption(row))
	}
	return options, nil
}

func (p productPostgresRepository) GetProductOptionByID(ctx context.Context, id uuid.UUID) (*ProductOption, error) {
	row, err := p.queries.GetProductOptionByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return mapOption(row), nil
}

func (p productPostgresRepository) UpdateProductOption(ctx context.Context, option *ProductOption) (*ProductOption, error) {
	row, err := p.queries.UpdateProductOption(ctx, productDb.UpdateProductOptionParams{
		ID:       option.ID,
		Name:     option.Name,
		Position: option.Position,
		IsActive: option.IsActive,
	})
	if err != nil {
		return nil, err
	}
	return mapOption(row), nil
}

func (p productPostgresRepository) DeleteProductOption(ctx context.Context, id uuid.UUID) error {
	result, err := p.queries.DeleteProductOption(ctx, id)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrOptionNotFound
	}
	return nil
}

// -------------------------------------------------------
// Option Values
// -------------------------------------------------------

func (p productPostgresRepository) CreateProductOptionValue(ctx context.Context, value *ProductOptionValue) (*ProductOptionValue, error) {
	row, err := p.queries.CreateProductOptionValue(ctx, productDb.CreateProductOptionValueParams{
		ID:              value.ID,
		ProductOptionID: value.ProductOptionID,
		Value:           value.Value,
		Position:        value.Position,
		IsActive:        value.IsActive,
	})
	if err != nil {
		return nil, err
	}
	return mapOptionValue(row), nil
}

func (p productPostgresRepository) GetProductOptionValues(ctx context.Context, optionID uuid.UUID) ([]*ProductOptionValue, error) {
	rows, err := p.queries.GetProductOptionValuesByOptionID(ctx, optionID)
	if err != nil {
		return nil, err
	}
	values := make([]*ProductOptionValue, 0, len(rows))
	for _, row := range rows {
		values = append(values, mapOptionValue(row))
	}
	return values, nil
}

func (p productPostgresRepository) DeleteProductOptionValue(ctx context.Context, id uuid.UUID) error {
	result, err := p.queries.DeleteProductOptionValue(ctx, id)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrOptionValueNotFound
	}
	return nil
}

// -------------------------------------------------------
// Images
// -------------------------------------------------------

func (p productPostgresRepository) CreateProductImage(ctx context.Context, image *ProductImage) (*ProductImage, error) {
	row, err := p.queries.CreateProductImage(ctx, productDb.CreateProductImageParams{
		ID:        image.ID,
		ProductID: image.ProductID,
		Url:       image.Url,
		AltText:   image.AltText,
		Position:  image.Position,
		IsPrimary: image.IsPrimary,
	})
	if dberrors.IsForeignKeyViolation(err) {
		return nil, ErrProductNotFound
	}
	if err != nil {
		return nil, err
	}
	return mapProductImage(row), nil
}

func (p productPostgresRepository) GetProductImages(ctx context.Context, productID uuid.UUID) ([]*ProductImage, error) {
	rows, err := p.queries.GetProductImagesByProductID(ctx, productID)
	if err != nil {
		return nil, err
	}
	images := make([]*ProductImage, 0, len(rows))
	for _, row := range rows {
		images = append(images, mapProductImage(row))
	}
	return images, nil
}

func (p productPostgresRepository) GetProductImageByID(ctx context.Context, id uuid.UUID) (*ProductImage, error) {
	row, err := p.queries.GetProductImageByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrProductImageNotFound
		}
		return nil, err
	}
	return mapProductImage(row), nil
}

func (p productPostgresRepository) SetPrimaryImage(ctx context.Context, productID uuid.UUID, imageID uuid.UUID) error {
	_, err := p.queries.SetPrimaryImage(ctx, productDb.SetPrimaryImageParams{
		ProductID: productID,
		ID:        imageID,
	})
	return err
}

func (p productPostgresRepository) DeleteProductImage(ctx context.Context, id uuid.UUID) error {
	result, err := p.queries.DeleteProductImage(ctx, id)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrProductImageNotFound
	}
	return nil
}

// -------------------------------------------------------
// Mapping helpers
// -------------------------------------------------------

func mapOption(row productDb.ProductOption) *ProductOption {
	return &ProductOption{
		ID:        row.ID,
		ProductID: row.ProductID,
		Name:      row.Name,
		Position:  row.Position,
		IsActive:  row.IsActive,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
		DeletedAt: row.DeletedAt,
	}
}

func mapOptionValue(row productDb.ProductOptionValue) *ProductOptionValue {
	return &ProductOptionValue{
		ID:              row.ID,
		ProductOptionID: row.ProductOptionID,
		Value:           row.Value,
		Position:        row.Position,
		IsActive:        row.IsActive,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
		DeletedAt:       row.DeletedAt,
	}
}

func mapProductImage(row productDb.ProductImage) *ProductImage {
	return &ProductImage{
		ID:        row.ID,
		ProductID: row.ProductID,
		Url:       row.Url,
		AltText:   row.AltText,
		Position:  row.Position,
		IsPrimary: row.IsPrimary,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
}

// -------------------------------------------------------
// Variants
// -------------------------------------------------------

func (p productPostgresRepository) CreateProductVariant(ctx context.Context, variant *ProductVariant, optionValueIDs []uuid.UUID) (*ProductVariantDetail, error) {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	q := productDb.New(tx)

	row, err := q.CreateProductVariant(ctx, productDb.CreateProductVariantParams{
		ID:        variant.ID,
		ProductID: variant.ProductID,
		ImageID:   variant.ImageID,
		Sku:       variant.Sku,
		Price:     variant.Price,
		Stock:     variant.Stock,
		Weight:    variant.Weight,
		Status:    productDb.VariantStatus(variant.Status),
	})
	if dberrors.IsForeignKeyViolation(err) {
		return nil, ErrProductNotFound
	}
	if dberrors.IsUniqueViolation(err) {
		return nil, ErrSkuAlreadyExists
	}
	if err != nil {
		return nil, err
	}

	if len(optionValueIDs) > 0 {
		err = q.BulkCreateProductVariantOptionValues(ctx, productDb.BulkCreateProductVariantOptionValuesParams{
			Column1: row.ID,
			Column2: optionValueIDs,
		})
		if dberrors.IsForeignKeyViolation(err) {
			return nil, ErrOptionValueNotFound
		}
		if err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return p.GetProductVariantByID(ctx, row.ID)
}

func (p productPostgresRepository) GetProductVariantByID(ctx context.Context, id uuid.UUID) (*ProductVariantDetail, error) {
	rows, err := p.queries.GetProductVariantByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, ErrVariantNotFound
	}

	first := rows[0]
	detail := &ProductVariantDetail{
		ProductVariant: ProductVariant{
			ID:        first.ID,
			ProductID: first.ProductID,
			ImageID:   first.ImageID,
			Sku:       first.Sku,
			Price:     first.Price,
			Stock:     first.Stock,
			Weight:    first.Weight,
			Status:    VariantStatus(first.Status),
			CreatedAt: first.CreatedAt,
			UpdatedAt: first.UpdatedAt,
			DeletedAt: first.DeletedAt,
		},
		OptionValues: make([]*ProductOptionValue, 0),
	}

	for _, row := range rows {
		if !row.ValueID.Valid {
			continue
		}
		detail.OptionValues = append(detail.OptionValues, &ProductOptionValue{
			ID:              row.ValueID.UUID,
			ProductOptionID: row.ProductOptionID.UUID,
			Value:           row.ValueText.String,
			Position:        row.ValuePosition.Int16,
			IsActive:        row.ValueIsActive.Bool,
			CreatedAt:       row.ValueCreatedAt.Time,
			UpdatedAt:       row.ValueUpdatedAt.Time,
		})
	}

	return detail, nil
}

func (p productPostgresRepository) GetProductVariants(ctx context.Context, productID uuid.UUID) ([]*ProductVariantDetail, error) {
	rows, err := p.queries.GetProductVariantsByProductID(ctx, productID)
	if err != nil {
		return nil, err
	}

	variantsMap := make(map[uuid.UUID]*ProductVariantDetail)
	variantsList := make([]*ProductVariantDetail, 0)

	for _, row := range rows {
		var detail *ProductVariantDetail
		var ok bool
		if detail, ok = variantsMap[row.ID]; !ok {
			detail = &ProductVariantDetail{
				ProductVariant: ProductVariant{
					ID:        row.ID,
					ProductID: row.ProductID,
					ImageID:   row.ImageID,
					Sku:       row.Sku,
					Price:     row.Price,
					Stock:     row.Stock,
					Weight:    row.Weight,
					Status:    VariantStatus(row.Status),
					CreatedAt: row.CreatedAt,
					UpdatedAt: row.UpdatedAt,
					DeletedAt: row.DeletedAt,
				},
				OptionValues: make([]*ProductOptionValue, 0),
			}
			variantsMap[row.ID] = detail
			variantsList = append(variantsList, detail)
		}

		if row.ValueID.Valid {
			detail.OptionValues = append(detail.OptionValues, &ProductOptionValue{
				ID:              row.ValueID.UUID,
				ProductOptionID: row.ProductOptionID.UUID,
				Value:           row.ValueText.String,
				Position:        row.ValuePosition.Int16,
				IsActive:        row.ValueIsActive.Bool,
				CreatedAt:       row.ValueCreatedAt.Time,
				UpdatedAt:       row.ValueUpdatedAt.Time,
			})
		}
	}

	return variantsList, nil
}

func (p productPostgresRepository) UpdateProductVariant(ctx context.Context, variant *ProductVariant) (*ProductVariantDetail, error) {
	row, err := p.queries.UpdateProductVariant(ctx, productDb.UpdateProductVariantParams{
		ID:      variant.ID,
		ImageID: variant.ImageID,
		Sku:     variant.Sku,
		Price:   variant.Price,
		Stock:   variant.Stock,
		Weight:  variant.Weight,
		Status:  productDb.VariantStatus(variant.Status),
	})
	if dberrors.IsUniqueViolation(err) {
		return nil, ErrSkuAlreadyExists
	}
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrVariantNotFound
		}
		return nil, err
	}

	return p.GetProductVariantByID(ctx, row.ID)
}

func (p productPostgresRepository) DeleteProductVariant(ctx context.Context, id uuid.UUID) error {
	result, err := p.queries.DeleteProductVariant(ctx, id)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrVariantNotFound
	}
	return nil
}

func mapProductVariant(row productDb.ProductVariant) *ProductVariant {
	return &ProductVariant{
		ID:        row.ID,
		ProductID: row.ProductID,
		ImageID:   row.ImageID,
		Sku:       row.Sku,
		Price:     row.Price,
		Stock:     row.Stock,
		Weight:    row.Weight,
		Status:    VariantStatus(row.Status),
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
		DeletedAt: row.DeletedAt,
	}
}
