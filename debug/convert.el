(require 'cl)

(defun replace-in-string (what with in)
  (replace-regexp-in-string (regexp-quote what) with in nil 'literal))


(defun gphp-migrate-fn (start end)
  "Common replaces php => gphp"
  (interactive "r")
  (kill-new
	   (gphp-replaces (buffer-substring-no-properties start end)
			  '(("private function" "func (p *Parser) ")
			    ("->parent =" ".P =")
			    ("TokenKind::" "lexer.")
			    ("TokenStringMaps::" "lexer.")
			    ("$this->" "p.")
			    ("p.getCurrentToken()" "p.token")
			    ("->" ".")
			    ("!==" "!=")
			    ("===" "==")
			    ("ParseContext::" "")
			    ("token.kind" "token.Kind")
			    ("= new " ":= ast.")
			    ("$" "")))))


(defun gphp-replaces (str list)
  (let (value)
    (setq value str)
    (dolist (elt list value)
      (setq value (replace-in-string (car elt) (cadr elt) value)))))
  


