package model

type CategoryType string

const (
	CategoryTypeIncome  CategoryType = "income"
	CategoryTypeOutcome CategoryType = "outcome"
)

type CategoryFrequency string

const (
	CategoryFrequencyFixed    CategoryFrequency = "fixed"
	CategoryFrequencyVariable CategoryFrequency = "variable"
)

type Category struct {
	Name             string            `json:"name"`
	Type             CategoryType      `json:"type"`
	Icon             string            `json:"icon"`
	ApplicableToFreq CategoryFrequency `json:"applicableToFreq"`
}

type CategoryName string

const (
	CategoryPrimaryIncome    CategoryName = "Primary Income"
	CategorySideIncome       CategoryName = "Side Income"
	CategoryInvestments      CategoryName = "Investments"
	CategoryOtherIncome      CategoryName = "Other Income"
	CategoryHousingUtilities CategoryName = "Housing & Utilities"
	CategoryFoodGroceries    CategoryName = "Food & Groceries"
	CategoryTransportation   CategoryName = "Transportation"
	CategoryHealthFitness    CategoryName = "Health & Fitness"
	CategoryEducation        CategoryName = "Education"
	CategoryEntertainment    CategoryName = "Entertainment & Travel"
	CategoryTaxesFees        CategoryName = "Taxes & Fees"
	CategoryPetsMisc         CategoryName = "Pets & Misc"
)

var Categories = []Category{
	{Name: string(CategoryPrimaryIncome), Type: CategoryTypeIncome, Icon: "💰", ApplicableToFreq: CategoryFrequencyFixed},
	{Name: string(CategorySideIncome), Type: CategoryTypeIncome, Icon: "💼", ApplicableToFreq: CategoryFrequencyVariable},
	{Name: string(CategoryInvestments), Type: CategoryTypeIncome, Icon: "📈", ApplicableToFreq: CategoryFrequencyVariable},
	{Name: string(CategoryOtherIncome), Type: CategoryTypeIncome, Icon: "🎁", ApplicableToFreq: CategoryFrequencyVariable},
	{Name: string(CategoryHousingUtilities), Type: CategoryTypeOutcome, Icon: "🏠", ApplicableToFreq: CategoryFrequencyFixed},
	{Name: string(CategoryFoodGroceries), Type: CategoryTypeOutcome, Icon: "🍽️", ApplicableToFreq: CategoryFrequencyVariable},
	{Name: string(CategoryTransportation), Type: CategoryTypeOutcome, Icon: "🚗", ApplicableToFreq: CategoryFrequencyVariable},
	{Name: string(CategoryHealthFitness), Type: CategoryTypeOutcome, Icon: "🏥", ApplicableToFreq: CategoryFrequencyVariable},
	{Name: string(CategoryEducation), Type: CategoryTypeOutcome, Icon: "📚", ApplicableToFreq: CategoryFrequencyVariable},
	{Name: string(CategoryEntertainment), Type: CategoryTypeOutcome, Icon: "🎉", ApplicableToFreq: CategoryFrequencyVariable},
	{Name: string(CategoryTaxesFees), Type: CategoryTypeOutcome, Icon: "📋", ApplicableToFreq: CategoryFrequencyFixed},
	{Name: string(CategoryPetsMisc), Type: CategoryTypeOutcome, Icon: "🐾", ApplicableToFreq: CategoryFrequencyVariable},
}

var categoryByName = make(map[CategoryName]Category)

func init() {
	for _, cat := range Categories {
		categoryByName[CategoryName(cat.Name)] = cat
	}
}

func GetCategoryByName(name string) (Category, bool) {
	cat, ok := categoryByName[CategoryName(name)]
	return cat, ok
}

func IsValidCategoryName(name string) bool {
	_, ok := categoryByName[CategoryName(name)]
	return ok
}

func GetAllCategories() []Category {
	return Categories
}

func GetCategoriesByType(categoryType CategoryType) []Category {
	var result []Category
	for _, cat := range Categories {
		if cat.Type == categoryType {
			result = append(result, cat)
		}
	}
	return result
}

func GetCategoriesByFrequency(frequency CategoryFrequency) []Category {
	var result []Category
	for _, cat := range Categories {
		if cat.ApplicableToFreq == frequency {
			result = append(result, cat)
		}
	}
	return result
}
