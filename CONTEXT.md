# Catalog

The catalog organizes fishing goods so customers can discover specific purchasable choices.

## Language

**Category**:
A node in the catalog taxonomy with a draft, active, or retired lifecycle. An active Category has only active ancestors; Products may be assigned to multiple active leaf Categories and inherit membership in each Category's ancestors. Only an unused draft Category may be deleted.

**Leaf Category**:
A Category with no children. It is the only level to which a Product may be assigned directly.

**Product**:
A catalog entry that groups one or more purchasable Variants. A Product is not itself purchased or stock-tracked; activation requires an active Brand, an active Variant, an active leaf Category assignment, and a Primary Product Image.

**Brand**:
The required commercial identity associated with a Product, with a draft, active, or inactive lifecycle. Only an unreferenced draft Brand may be deleted.

**Variant**:
A specific purchasable form of a Product that owns its SKU, Price, Stock, and Weight. An active non-default Variant selects exactly one Value from every active Option; a Product may also offer one default Variant with no selections. Each active Option Value combination identifies at most one active Variant within a Product.

**Option**:
A case-insensitively unique variation dimension within a Product, such as Size or Color. Each non-default Variant selects exactly one of the Option's Values; Options and Values change only while the Product is not active.

**Option Value**:
One permitted choice for an Option, such as Medium for Size. Values are case-insensitively unique within an Option and may be selected only by Variants of the same Product.

**Stock**:
The nonnegative quantity currently available for a Variant. Zero means the active Variant remains visible but is sold out.

**Price**:
The whole-rupiah amount charged for a Variant, denominated in the catalog-wide currency `IDR`. An active Variant's Price is greater than zero.

**Weight**:
The positive whole-gram shipping weight of an active Variant. A draft Variant may omit Weight.

**SKU**:
The case-insensitively unique, durable business identifier of a Variant. It may change while the Variant is a draft and becomes immutable after activation.

**Slug**:
The durable public identifier of a Brand, Category, or Product. Category Slugs are globally unique across the taxonomy; every Slug becomes immutable after first publication and is never reassigned to another record.

**Catalog Administrator**:
An authenticated person permitted to create, change, retire, and delete catalog content. Catalog reads remain public.

**Archive**:
Remove a previously active Product or Variant from sale while preserving its history. Archiving a Product makes all its Variants unavailable without changing their individual statuses.

**Delete**:
Discard a Product or Variant that remained a draft and was never active. Published Products and Variants are archived instead.

**Inactive Brand**:
A retired Brand retained on existing active Products but unavailable for new Product assignments. Its retirement does not make those Products unsellable, but draft Products referencing it must select an active Brand before activation.

**Catalog Image**:
An image owned and managed by the catalog for a Brand or Product. Arbitrary externally hosted image URLs are not Catalog Images.

**Product Image**:
A Catalog Image in a Product's gallery. Variants may reference an image only from their own Product; if that image is removed, they fall back to the Primary Product Image.

**Primary Product Image**:
The single default image for a Product. Every Product with a nonempty image gallery has exactly one.
