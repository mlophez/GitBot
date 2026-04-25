package app

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"gitbot/internal/logger"
)

// ValidateAppRequest is the subset of the Kubernetes AdmissionReview request
// body that this handler reads.
type ValidateAppRequest struct {
	Request struct {
		UID       string `json:"uid"`
		Name      string `json:"name"`
		Namespace string `json:"namespace"`
		Operation string `json:"operation"`
		UserInfo  struct {
			Username string `json:"username"`
		} `json:"userInfo"`
		OldObject struct {
			Metadata struct {
				Annotations map[string]string `json:"annotations"`
			} `json:"metadata"`
			Spec struct {
				Source struct {
					TargetRevision string `json:"targetRevision"`
				} `json:"source"`
			} `json:"spec"`
		} `json:"oldObject"`
		Object struct {
			Spec struct {
				Source struct {
					TargetRevision string `json:"targetRevision"`
				} `json:"source"`
			} `json:"spec"`
		} `json:"object"`
	} `json:"request"`
}

// ValidateAppResponse is the Kubernetes AdmissionReview response body.
type ValidateAppResponse struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Response   struct {
		UID     string `json:"uid"`
		Allowed bool   `json:"allowed"`
		Status  struct {
			Message string `json:"message"`
		} `json:"status"`
	} `json:"response"`
}

// ValidateApp handles POST /validate (Kubernetes ValidatingAdmissionWebhook).
// Allows or denies changes to an ArgoCD Application's targetRevision field.
// Returns 200 with allowed=false when a locked app's targetRevision is changed by a non-bot user.
// Returns 200 with allowed=true when: the change comes from the bot user, the targetRevision
// did not change, or the application is not locked.
func ValidateApp(botUsername string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.Logger(r.Context())

		var req ValidateAppRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Error("ValidateApp failed to decode request", "error", err)
			writeAdmissionResponse(w, log, "0000", false, "Server Error: invalid request body")
			return
		}

		uid := req.Request.UID
		name := req.Request.Name
		namespace := req.Request.Namespace
		operation := req.Request.Operation
		username := req.Request.UserInfo.Username
		newRevision := req.Request.Object.Spec.Source.TargetRevision
		// TODO: This need get app from appmanager and check if it's locked, instead of relying on the client to send the annotation in the admission request.
		// If annotation changed in future, this will cause a problem. We should get the app and check if it's locked in the admission controller, instead of relying on the client to send the annotation in the admission request.
		// But this can be produce side effect, because we need to get app from appmanager, and if the app is not exist, we will return error, but in this case, we should allow the change, because the app is not exist, so we can create it.
		oldRevision := req.Request.OldObject.Spec.Source.TargetRevision
		isLocked := req.Request.OldObject.Metadata.Annotations["bot.gitbot.io/locked"] == "true"

		commonFields := []any{
			"uid", uid,
			"user", username,
			"name", name,
			"namespace", namespace,
			"operation", operation,
			"oldTargetRevision", oldRevision,
			"newTargetRevision", newRevision,
		}

		if username == botUsername {
			log.Info("ValidateApp allowed: change from bot user", append(commonFields, "allowed", true)...)
			writeAdmissionResponse(w, log, uid, true, "Change allowed from '"+botUsername+"'")
			return
		}

		if oldRevision == newRevision {
			log.Info("ValidateApp allowed: targetRevision unchanged", append(commonFields, "allowed", true)...)
			writeAdmissionResponse(w, log, uid, true, "TargetRevision did not change")
			return
		}

		if !isLocked {
			log.Info("ValidateApp allowed: application is not locked", append(commonFields, "allowed", true)...)
			writeAdmissionResponse(w, log, uid, true, "Application is not locked")
			return
		}

		log.Info("ValidateApp denied: locked app targetRevision changed by non-bot user", append(commonFields, "allowed", false)...)
		writeAdmissionResponse(w, log, uid, false,
			"TargetRevision change denied: Application is locked, target revision changed and username is not '"+botUsername+"'",
		)
	}
}

// writeAdmissionResponse serialises and writes an AdmissionReview response.
// It always returns HTTP 200, as required by the Kubernetes admission webhook protocol.
func writeAdmissionResponse(w http.ResponseWriter, log *slog.Logger, uid string, allowed bool, message string) {
	var resp ValidateAppResponse
	resp.APIVersion = "admission.k8s.io/v1"
	resp.Kind = "AdmissionReview"
	resp.Response.UID = uid
	resp.Response.Allowed = allowed
	resp.Response.Status.Message = message

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Error("ValidateApp failed to write response", "error", err)
	}
}
