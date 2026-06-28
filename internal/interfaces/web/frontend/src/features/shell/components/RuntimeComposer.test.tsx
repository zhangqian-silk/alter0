import { fireEvent, render, screen } from "@testing-library/react";
import { RuntimeComposer } from "./RuntimeComposer";

function renderComposer(overrides: Partial<React.ComponentProps<typeof RuntimeComposer>> = {}) {
  const inputRef = { current: null as HTMLTextAreaElement | null };
  const onSubmit = vi.fn((event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
  });
  const onAddAttachment = vi.fn();
  const onUtilityClick = vi.fn();

  const result = render(
    <RuntimeComposer
      runtimeKind="chat"
      onSubmit={onSubmit}
      attachments={[]}
      attachmentPreviewLabel={(attachment) => `Preview ${attachment.name}`}
      attachmentRemoveLabel={(attachment) => `Remove ${attachment.name}`}
      previewAttachment={null}
      onPreviewAttachmentChange={vi.fn()}
      onRemoveAttachment={vi.fn()}
      inputLabel="Message"
      inputId="composer-test-input"
      inputRef={inputRef}
      inputValue=""
      onInputChange={vi.fn()}
      utilityButtons={[{
        key: "session",
        label: "Session",
        onClick: onUtilityClick,
      }]}
      addAttachmentLabel="Add attachment"
      onAddAttachment={onAddAttachment}
      submitLabel="Send"
      previewCloseLabel="Close preview"
      {...overrides}
    />,
  );

  return {
    ...result,
    input: inputRef.current as HTMLTextAreaElement,
    onAddAttachment,
    onSubmit,
    onUtilityClick,
  };
}

describe("RuntimeComposer", () => {
  it("blurs the active input before running utility toolbar actions", () => {
    const { input, onUtilityClick } = renderComposer();
    input.focus();
    const blur = vi.spyOn(input, "blur");

    fireEvent.click(screen.getByRole("button", { name: "Session" }));

    expect(blur).toHaveBeenCalledTimes(1);
    expect(onUtilityClick).toHaveBeenCalledTimes(1);
  });

  it("blurs the active input during touch capture before utility actions open panels", () => {
    const { input, onUtilityClick } = renderComposer();
    input.focus();
    const blur = vi.spyOn(input, "blur");

    fireEvent.touchStart(screen.getByRole("button", { name: "Session" }));

    expect(blur).toHaveBeenCalledTimes(1);
    expect(onUtilityClick).not.toHaveBeenCalled();
  });

  it("blurs the active input before running attachment and submit actions", () => {
    const { input, onAddAttachment, onSubmit } = renderComposer();
    input.focus();
    const blur = vi.spyOn(input, "blur");

    fireEvent.click(screen.getByRole("button", { name: "Add attachment" }));
    input.focus();
    fireEvent.click(screen.getByRole("button", { name: "Send" }));

    expect(blur).toHaveBeenCalledTimes(2);
    expect(onAddAttachment).toHaveBeenCalledTimes(1);
    expect(onSubmit).toHaveBeenCalledTimes(1);
  });
});
