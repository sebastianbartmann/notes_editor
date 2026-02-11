package api

import "net/http"

const manualCommitMessage = "manual app commit"

type GitStatusResponse struct {
	Output string `json:"output"`
}

type GitActionResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Output  string `json:"output,omitempty"`
}

func (s *Server) handleGitStatus(w http.ResponseWriter, r *http.Request) {
	if _, ok := requirePerson(w, r); !ok {
		return
	}

	s.mu.Lock()
	out, err := s.git.StatusShort()
	s.mu.Unlock()
	if err != nil {
		writeBadRequest(w, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, GitStatusResponse{Output: out})
}

func (s *Server) handleGitCommit(w http.ResponseWriter, r *http.Request) {
	if _, ok := requirePerson(w, r); !ok {
		return
	}

	s.mu.Lock()
	committed, err := s.git.Commit(manualCommitMessage)
	statusOut, statusErr := s.git.StatusShort()
	s.mu.Unlock()
	if err != nil {
		writeBadRequest(w, err.Error())
		return
	}
	if statusErr != nil {
		statusOut = ""
	}

	msg := "No changes to commit"
	if committed {
		msg = "Committed changes"
	}
	writeJSON(w, http.StatusOK, GitActionResponse{
		Success: true,
		Message: msg,
		Output:  statusOut,
	})
}

func (s *Server) handleGitPush(w http.ResponseWriter, r *http.Request) {
	if _, ok := requirePerson(w, r); !ok {
		return
	}

	s.mu.Lock()
	err := s.git.Push()
	statusOut, statusErr := s.git.StatusShort()
	s.mu.Unlock()
	if err != nil {
		s.syncMgr.RecordManualPush(err)
		writeBadRequest(w, err.Error())
		return
	}
	s.syncMgr.RecordManualPush(nil)
	if statusErr != nil {
		statusOut = ""
	}

	writeJSON(w, http.StatusOK, GitActionResponse{
		Success: true,
		Message: "Pushed changes",
		Output:  statusOut,
	})
}

func (s *Server) handleGitPull(w http.ResponseWriter, r *http.Request) {
	if _, ok := requirePerson(w, r); !ok {
		return
	}

	s.mu.Lock()
	err := s.git.PullFFOnly()
	statusOut, statusErr := s.git.StatusShort()
	s.mu.Unlock()
	if err != nil {
		s.syncMgr.RecordManualPull(err)
		writeBadRequest(w, "Pull failed (ff-only). Resolve divergence/conflicts first: "+err.Error())
		return
	}
	s.syncMgr.RecordManualPull(nil)
	if statusErr != nil {
		statusOut = ""
	}

	writeJSON(w, http.StatusOK, GitActionResponse{
		Success: true,
		Message: "Pulled latest changes",
		Output:  statusOut,
	})
}

func (s *Server) handleGitCommitPush(w http.ResponseWriter, r *http.Request) {
	if _, ok := requirePerson(w, r); !ok {
		return
	}

	s.mu.Lock()
	committed, commitErr := s.git.Commit(manualCommitMessage)
	if commitErr != nil {
		s.mu.Unlock()
		writeBadRequest(w, commitErr.Error())
		return
	}
	pushErr := s.git.Push()
	statusOut, statusErr := s.git.StatusShort()
	s.mu.Unlock()

	if pushErr != nil {
		s.syncMgr.RecordManualPush(pushErr)
		writeBadRequest(w, pushErr.Error())
		return
	}
	s.syncMgr.RecordManualPush(nil)
	if statusErr != nil {
		statusOut = ""
	}

	msg := "Pushed changes"
	if committed {
		msg = "Committed and pushed changes"
	}
	writeJSON(w, http.StatusOK, GitActionResponse{
		Success: true,
		Message: msg,
		Output:  statusOut,
	})
}
