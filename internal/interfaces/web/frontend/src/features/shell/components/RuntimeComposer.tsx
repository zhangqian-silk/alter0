import {
  type ChangeEvent,
  type ComponentPropsWithoutRef,
  type FocusEventHandler,
  type FormEvent,
  type PointerEventHandler,
  type ReactNode,
  type Ref,
  type TouchEventHandler,
} from "react";
import {
  resolveComposerAttachmentPreviewURL,
  type ComposerImageAttachment,
} from "../../conversation-runtime/composerImageAttachments";

function joinClassNames(...values: Array<string | undefined>) {
  return values.filter(Boolean).join(" ");
}

type RuntimeComposerProps = {
  shellRef?: Ref<HTMLElement>;
  shellClassName?: string;
  shellProps?: Omit<ComponentPropsWithoutRef<"footer">, "children" | "className"> & {
    className?: string;
  };
  note?: ReactNode;
  noteClassName?: string;
  noteProps?: Omit<ComponentPropsWithoutRef<"div">, "children" | "className"> & {
    className?: string;
  };
  formProps?: Omit<ComponentPropsWithoutRef<"form">, "children" | "className" | "onSubmit"> & {
    className?: string;
  };
  onSubmit: (event: FormEvent<HTMLFormElement>) => void;
  fileInputRef?: Ref<HTMLInputElement>;
  onFileChange: (event: ChangeEvent<HTMLInputElement>) => void;
  attachments: ComposerImageAttachment[];
  attachmentStripProps?: Omit<ComponentPropsWithoutRef<"div">, "children" | "className"> & {
    className?: string;
  };
  attachmentPreviewLabel: (attachment: ComposerImageAttachment) => string;
  attachmentRemoveLabel: (attachment: ComposerImageAttachment) => string;
  previewAttachment: ComposerImageAttachment | null;
  onPreviewAttachmentChange: (attachment: ComposerImageAttachment | null) => void;
  onRemoveAttachment: (attachment: ComposerImageAttachment) => void;
  inputLabel: string;
  inputId: string;
  inputRef?: Ref<HTMLTextAreaElement>;
  inputValue: string;
  inputProps?: Omit<ComponentPropsWithoutRef<"textarea">, "id" | "ref" | "children" | "className" | "value" | "onChange"> & {
    className?: string;
  };
  onInputChange: (value: string) => void;
  onInputFocus?: FocusEventHandler<HTMLTextAreaElement>;
  onInputBlur?: FocusEventHandler<HTMLTextAreaElement>;
  onInputPointerDownCapture?: PointerEventHandler<HTMLTextAreaElement>;
  onInputTouchStartCapture?: TouchEventHandler<HTMLTextAreaElement>;
  toolsClassName?: string;
  metaClassName?: string;
  metaContent?: ReactNode;
  metaProps?: Omit<ComponentPropsWithoutRef<"div">, "children" | "className"> & {
    className?: string;
  };
  addImageLabel: string;
  addImageButtonProps?: Omit<ComponentPropsWithoutRef<"button">, "type" | "children" | "onClick" | "aria-label" | "className"> & {
    className?: string;
  };
  onAddImage: () => void;
  submitButtonProps?: Omit<ComponentPropsWithoutRef<"button">, "type" | "children" | "aria-label" | "className"> & {
    className?: string;
  };
  submitLabel: string;
  submitIcon: ReactNode;
  previewCloseLabel: string;
};

export function RuntimeComposer({
  shellRef,
  shellClassName,
  shellProps,
  note,
  noteClassName,
  noteProps,
  formProps,
  onSubmit,
  fileInputRef,
  onFileChange,
  attachments,
  attachmentStripProps,
  attachmentPreviewLabel,
  attachmentRemoveLabel,
  previewAttachment,
  onPreviewAttachmentChange,
  onRemoveAttachment,
  inputLabel,
  inputId,
  inputRef,
  inputValue,
  inputProps,
  onInputChange,
  onInputFocus,
  onInputBlur,
  onInputPointerDownCapture,
  onInputTouchStartCapture,
  toolsClassName,
  metaClassName,
  metaContent,
  metaProps,
  addImageLabel,
  addImageButtonProps,
  onAddImage,
  submitButtonProps,
  submitLabel,
  submitIcon,
  previewCloseLabel,
}: RuntimeComposerProps) {
  const {
    className: shellPropsClassName,
    ...shellRestProps
  } = shellProps || {};
  const {
    className: notePropsClassName,
    ...noteRestProps
  } = noteProps || {};
  const {
    className: formPropsClassName,
    ...formRestProps
  } = formProps || {};
  const {
    className: attachmentStripPropsClassName,
    ...attachmentStripRestProps
  } = attachmentStripProps || {};
  const {
    className: inputPropsClassName,
    ...inputRestProps
  } = inputProps || {};
  const {
    className: metaPropsClassName,
    ...metaRestProps
  } = metaProps || {};
  const {
    className: addImageButtonPropsClassName,
    ...addImageButtonRestProps
  } = addImageButtonProps || {};
  const {
    className: submitButtonPropsClassName,
    ...submitButtonRestProps
  } = submitButtonProps || {};

  return (
    <>
      <footer
        ref={shellRef}
        className={joinClassNames(
          "runtime-composer-shell",
          shellClassName,
          shellPropsClassName,
        )}
        {...shellRestProps}
      >
        {note ? (
          <div
            className={joinClassNames("runtime-composer-note", noteClassName, notePropsClassName)}
            {...noteRestProps}
          >
            {note}
          </div>
        ) : null}
        <form
          className={joinClassNames(
            "runtime-composer-form",
            formPropsClassName,
          )}
          data-runtime-composer="true"
          onSubmit={onSubmit}
          {...formRestProps}
        >
          <input
            ref={fileInputRef}
            type="file"
            accept="image/*"
            hidden
            multiple
            onChange={onFileChange}
          />
          {attachments.length > 0 ? (
            <div
              className={joinClassNames(
                "runtime-composer-attachments",
                attachmentStripPropsClassName,
              )}
              data-runtime-attachment-strip="true"
              {...attachmentStripRestProps}
            >
              {attachments.map((attachment) => (
                <article key={attachment.id} className="runtime-composer-attachment">
                  <button
                    type="button"
                    className="runtime-composer-attachment-preview"
                    aria-label={attachmentPreviewLabel(attachment)}
                    onClick={() => onPreviewAttachmentChange(attachment)}
                  >
                    <img
                      src={resolveComposerAttachmentPreviewURL(attachment)}
                      alt={attachment.name}
                      loading="lazy"
                      decoding="async"
                    />
                  </button>
                  <button
                    type="button"
                    className="runtime-composer-attachment-remove"
                    aria-label={attachmentRemoveLabel(attachment)}
                    onClick={() => onRemoveAttachment(attachment)}
                  >
                    ×
                  </button>
                </article>
              ))}
            </div>
          ) : null}
          <label className="sr-only" htmlFor={inputId}>
            {inputLabel}
          </label>
          <textarea
            id={inputId}
            ref={inputRef}
            className={joinClassNames(
              "runtime-composer-input",
              inputPropsClassName,
            )}
            value={inputValue}
            onPointerDownCapture={onInputPointerDownCapture}
            onTouchStartCapture={onInputTouchStartCapture}
            onChange={(event) => onInputChange(event.target.value)}
            onFocus={onInputFocus}
            onBlur={onInputBlur}
            {...inputRestProps}
          ></textarea>
          <div
            className={joinClassNames(
              "runtime-composer-tools",
              toolsClassName,
            )}
          >
            <div
              className={joinClassNames(
                "runtime-composer-meta",
                metaClassName,
                metaPropsClassName,
              )}
              {...metaRestProps}
            >
              {metaContent}
            </div>
            <button
              type="button"
              className={joinClassNames(
                "runtime-composer-upload",
                addImageButtonPropsClassName,
              )}
              aria-label={addImageLabel}
              onClick={onAddImage}
              {...addImageButtonRestProps}
            >
              <span aria-hidden="true">+</span>
              <span>{addImageLabel}</span>
            </button>
            <button
              type="submit"
              className={joinClassNames(
                "runtime-composer-submit",
                submitButtonPropsClassName,
              )}
              aria-label={submitLabel}
              {...submitButtonRestProps}
            >
              <span className="runtime-composer-submit-icon" aria-hidden="true">
                {submitIcon}
              </span>
              <span className="sr-only">{submitLabel}</span>
            </button>
          </div>
        </form>
      </footer>
      {previewAttachment ? (
        <div
          className="runtime-image-preview-backdrop"
          data-runtime-attachment-preview="true"
          onClick={() => onPreviewAttachmentChange(null)}
        >
          <div
            className="runtime-image-preview-dialog"
            role="dialog"
            aria-modal="true"
            aria-label={previewAttachment.name}
            onClick={(event) => event.stopPropagation()}
          >
            <button
              type="button"
              className="runtime-image-preview-close"
              aria-label={previewCloseLabel}
              onClick={() => onPreviewAttachmentChange(null)}
            >
              ×
            </button>
            <img src={resolveComposerAttachmentPreviewURL(previewAttachment)} alt={previewAttachment.name} decoding="async" />
          </div>
        </div>
      ) : null}
    </>
  );
}
