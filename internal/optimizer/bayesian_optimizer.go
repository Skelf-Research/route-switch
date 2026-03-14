package optimizer

import (
	"fmt"

	"github.com/c-bata/goptuna"
)

// GoptunaBayesianOptimizer wraps the goptuna library to implement our BayesianOptimizer interface
type GoptunaBayesianOptimizer struct {
	study *goptuna.Study
	config map[string]interface{}
}

// NewGoptunaBayesianOptimizer creates a new instance using goptuna
func NewGoptunaBayesianOptimizer(config map[string]interface{}) (*GoptunaBayesianOptimizer, error) {
	// Create a study with TPE sampler (Tree-structured Parzen Estimator)
	study, err := goptuna.CreateStudy(
		"prompt-optimization",
		goptuna.StudyOptionSampler(goptuna.NewRandomSampler()),
		goptuna.StudyOptionDirection(goptuna.StudyDirectionMaximize),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create study: %w", err)
	}

	return &GoptunaBayesianOptimizer{
		study:  study,
		config: config,
	}, nil
}

// Optimize performs Bayesian optimization on the search space
func (g *GoptunaBayesianOptimizer) Optimize(searchSpace map[string]interface{}, objectiveFn func(params map[string]interface{}) (float64, error)) (map[string]interface{}, float64, error) {
	// Determine number of trials from config or use default
	nTrials := 10 // default
	if val, ok := g.config["num_trials"].(int); ok {
		nTrials = val
	}

	// Define the objective function that goptuna will optimize
	objective := func(trial goptuna.Trial) (float64, error) {
		// Extract parameters based on search space definition
		params := make(map[string]interface{})
		
		for key, value := range searchSpace {
			switch v := value.(type) {
			case map[string]interface{}:
				// Handle different parameter types based on 'type' field
				if paramType, ok := v["type"].(string); ok {
					switch paramType {
					case "categorical":
						if choices, ok := v["choices"].([]interface{}); ok {
							choiceIndex, err := trial.SuggestInt(key, 0, len(choices)-1)
							if err != nil {
								return 0, err
							}
							params[key] = choices[choiceIndex]
						}
					case "float":
						if low, ok := v["low"].(float64); ok {
							if high, ok := v["high"].(float64); ok {
								suggestion, err := trial.SuggestFloat(key, low, high)
								if err != nil {
									return 0, err
								}
								params[key] = suggestion
							}
						}
					case "int":
						if low, ok := v["low"].(int); ok {
							if high, ok := v["high"].(int); ok {
								suggestion, err := trial.SuggestInt(key, low, high)
								if err != nil {
									return 0, err
								}
								params[key] = suggestion
							}
						}
					}
				}
			}
		}
		
		// Call the objective function with these parameters
		score, err := objectiveFn(params)
		if err != nil {
			return 0, err
		}
		
		// Since goptuna minimizes by default and we want to maximize score,
		// we return the negative score for maximization
		return -score, nil
	}

	// Optimize using goptuna
	err := g.study.Optimize(objective, nTrials)
	if err != nil {
		return nil, 0, fmt.Errorf("optimization failed: %w", err)
	}

	// Get the best parameters
	bestParams, err := g.study.GetBestParams()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get best parameters: %w", err)
	}

	// Get the best value (this is the negative of our actual score since we negated it)
	bestValue, err := g.study.GetBestValue()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get best value: %w", err)
	}

	// Convert back to positive (since we negated for maximization)
	bestValue = -bestValue

	return bestParams, bestValue, nil
}

// Name returns the name of this optimizer
func (g *GoptunaBayesianOptimizer) Name() string {
	return "GoptunaBayesianOptimizer"
}