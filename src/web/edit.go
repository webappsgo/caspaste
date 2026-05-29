// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package web

import (
	"net/http"

	"github.com/casjay-forks/caspaste/src/netshare"
	"github.com/casjay-forks/caspaste/src/storage"
)

// POST /edit/{id} - Edit an editable paste
func (data *Data) handleEditPaste(rw http.ResponseWriter, req *http.Request) error {
	// Check method
	if req.Method != "POST" {
		return netshare.ErrMethodNotAllowed
	}

	// Get ID from path
	id := req.URL.Path[len("/edit/"):]

	// Check rate limit
	err := data.RateLimitNew.CheckAndUse(netshare.GetClientAddr(req))
	if err != nil {
		return err
	}

	// Get existing paste
	paste, err := data.DB.PasteGet(id)
	if err != nil {
		return err
	}

	// Check if paste is editable
	if !paste.IsEditable {
		return netshare.ErrUnauthorized
	}

	// Parse form
	req.ParseForm()

	// Update paste fields
	newBody := req.PostForm.Get("body")
	if newBody != "" {
		paste.Body = newBody
	}

	newTitle := req.PostForm.Get("title")
	if newTitle != "" {
		paste.Title = newTitle
	}

	// Update paste (need to add PasteUpdate method)
	updatedPaste := storage.Paste{
		ID:          paste.ID,
		Title:       paste.Title,
		Body:        paste.Body,
		Syntax:      paste.Syntax,
		CreateTime:  paste.CreateTime,
		DeleteTime:  paste.DeleteTime,
		OneUse:      paste.OneUse,
		Author:      paste.Author,
		AuthorEmail: paste.AuthorEmail,
		AuthorURL:   paste.AuthorURL,
		IsFile:      paste.IsFile,
		FileName:    paste.FileName,
		MimeType:    paste.MimeType,
		IsEditable:  paste.IsEditable,
		IsPrivate:   paste.IsPrivate,
		IsURL:       paste.IsURL,
		OriginalURL: paste.OriginalURL,
	}

	err = data.DB.PasteUpdate(updatedPaste)
	if err != nil {
		return err
	}

	// Redirect to paste page
	http.Redirect(rw, req, "/"+id, http.StatusSeeOther)
	return nil
}
