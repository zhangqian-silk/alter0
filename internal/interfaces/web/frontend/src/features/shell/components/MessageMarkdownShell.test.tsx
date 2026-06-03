import { fireEvent, render, screen } from "@testing-library/react";
import { useState } from "react";
import { MessageMarkdownShell } from "./MessageMarkdownShell";

describe("MessageMarkdownShell", () => {
  it("renders markdown through the shared selectable shell", () => {
    const { container } = render(
      <MessageMarkdownShell
        markdown={"# Message\n\nUse `pwd` before opening [Chat](/chat)."}
        copyValue="copy payload"
        copyLabel="Copy output"
        className="terminal-final-text"
        bodyClassName="terminal-final-rendered"
      />,
    );

    const shell = container.querySelector(".message-markdown-shell");
    expect(shell?.children.item(0)).toHaveClass("message-markdown-body");
    expect(shell?.children.item(1)).toHaveClass("message-markdown-toolbar");
    expect(container.querySelector(".terminal-final-rendered")).toContainHTML("<h1>Message</h1>");
    expect(container.querySelector(".chat-md-inline-code")).toHaveTextContent("pwd");
    expect(screen.getByRole("link", { name: "Chat" })).toHaveAttribute("href", "/chat");
  });

  it("keeps rendered text nodes stable across unrelated parent renders", () => {
    function Harness() {
      const [count, setCount] = useState(0);
      return (
        <>
          <button type="button" onClick={() => setCount((current) => current + 1)}>
            rerender {count}
          </button>
          <MessageMarkdownShell
            markdown="Hello. What should we work on?"
            copyValue="Hello. What should we work on?"
            copyLabel="Copy output"
          />
        </>
      );
    }

    const { container } = render(<Harness />);
    const paragraph = container.querySelector(".message-markdown-rendered p");

    fireEvent.click(screen.getByRole("button", { name: /rerender/i }));

    expect(container.querySelector(".message-markdown-rendered p")).toBe(paragraph);
  });
});
