# Iter8 Experiment API spec

This file is the documentation for the iter8 Experiment spec.

```yaml
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  creationTimestamp: null
  labels:
    controller-tools.k8s.io: "1.0"
  name: experiments.iter8.tools
spec:
  additionalPrinterColumns:
  - JSONPath: .status.conditions[?(@.type == 'ExperimentCompleted')].status
    # [True, False]
    description: Whether experiment is completed
    format: byte
    name: completed
    type: string
  - JSONPath: .status.conditions[?(@.type == 'Ready')].reason
    # [Experiment status messages like 'Progressing', 'Candidate missing', ...]
    description: Status of the experiment 
    format: byte
    name: status
    type: string
  - JSONPath: .spec.targetService.baseline
    description: Name of baseline
    format: byte
    name: baseline
    type: string
  - JSONPath: .status.trafficSplitPercentage.baseline
    description: Traffic percentage for baseline
    format: int32
    name: percentage
    type: integer
  - JSONPath: .spec.targetService.candidate
    description: Name of candidate
    format: byte
    name: candidate
    type: string
  - JSONPath: .status.trafficSplitPercentage.candidate
    description: Traffic percentage for candidate
    format: int32
    name: percentage
    type: integer
  group: iter8.tools
  names:
    categories:
    - all
    - iter8
    kind: Experiment
    plural: experiments
  scope: Namespaced
  subresources:
    status: {}
  validation:
    openAPIV3Schema:
      # Boiler plate for experiment object
      properties:
        apiVersion:
          type: string
        kind:
          type: string
        metadata:
          type: object
        spec:
          properties:
            analysis: # optional section
              properties:
                analyticsService: # optional; default value = http://iter8analytics:5555
                  type: string
                # The endpoint to grafana dashboard; 
                # optional; default value = http://localhost:3000
                grafanaEndpoint:
                  type: string
                # list of criteria which determines if experiment succeeds
                successCriteria: # optional section -- the controller can run without any successCriteria
                  items:
                    properties:
                      metricName:
                        enum:
                        - iter8_latency # mean latency of the service 
                        - iter8_error_rate # mean error rate (~5** HTTP Status codes) of the service
                        - iter8_error_count # total error count (~5** HTTP Status codes) of the service
                        type: string
                      sampleSize: # optional; default = 10
                        format: int64
                        type: integer
                      stopOnFailure: # optional; default = False
                        type: boolean
                      tolerance: # tolerance limit in the success criteria
                        format: double 
                        type: number
                      toleranceType:
                        enum:
                        - threshold # tolerance limit is an absolute value
                        - delta # tolerance limit is a relative value compared to baseline 
                        type: string
                    required:
                    - name
                    - tolerance
                    - toleranceType
                    type: object
                  type: array
              type: object
            targetService: # service for which experiment is performed
              properties: # This needs to be resolved
              # For istio, both of these are required
              # In knative, it seems neither of them are required.
                baseline:
                  type: string
                candidate:
                  type: string
              type: object
            trafficControl: # optional
              properties:
                # duration of an iteration
                interval: # optional; default = 1 min
                  type: string
                # maximum total number of iterations in the experiment
                maxIterations: # optional; default value = 100
                  format: int64
                  type: integer
                # maximum percentage of traffic routed to candidate during the experiment
                # in knative, the max value of this seems to be 99. Can we make this 100?
                maxTrafficPercentage: # optional; default value = 50; 
                  format: double
                  type: number
                # where should traffic go on successful completion of experiment?
                onSuccess: # optional; default value is candidate
                  enum:
                  - baseline
                  - candidate
                  - both
                  type: string
                # minimum percentage increase in traffic
                trafficStepSize: # optional; default value is 2.0 percent
                  format: double
                  type: number
                # What strategy to use for experiment.
                # Supported strategies are check_and_increment and increment_without_check.
                # increment_without_check is useful for testing purposes.
                strategy: # optional; default value is check_and_increment
                  enum:
                  - check_and_increment
                  _ increment_without_check
                  type: string
              type: object
          required:
          - targetService
          # - trafficControl # should be optional
          type: object
        status:
          properties:
            # This is the last state populated by the analytics service.
            # The analysisState is intended to be a blackbox to the controller and
            # interpreted only by the analytics service.
            analysisState:
              type: object
            # Plain text summary of the experiment
            # Experiment could have completed or could be still going on
            assessment: 
              properties:
                conclusions:
                  items:
                    type: string
                  type: array
              type: object
            # kubectl output shows the last change for each of the above conditions
            conditions: 
              items:
                properties:
                  lastTransitionTime:
                    type: string
                  message:
                    type: string
                  reason: # we will provide a reason only if status is false
                    type: string
                  severity:
                    type: string
                  status: # [True, False, Unknown]
                    type: string
                  type: #[ServiceProvided, ExperimentCompleted, RollForward, Ready]
                    type: string
                required:
                - type
                - status
                type: object
              type: array
            # number of iterations of the experiment that have finished so far 
            currentIteration: 
              format: int64
              type: integer
            # timestamp when experiment is completed
            endTimestamp:
              type: string
            # the url to the grafana dashboard
            grafanaURL:
              type: string
            # time at which the last iteration completed
            lastIterationCompletionTime: 
              format: date-time
              type: string
            # to be resolved... ???
            observedGeneration:
              format: int64
              type: integer
            # timestamp when experiment is started
            startTimestamp:
              type: string
            # current traffic split between baseline and candidate
            trafficSplitPercentage: 
              properties:
                baseline:
                  format: int64
                  type: integer
                candidate:
                  format: int64
                  type: integer
              required:
              - baseline
              - candidate
              type: object
          type: object
  version: v1alpha1
# to be resolved... ???
status:
  acceptedNames:
    kind: ""
    plural: ""
  conditions: []
  storedVersions: []
  ```