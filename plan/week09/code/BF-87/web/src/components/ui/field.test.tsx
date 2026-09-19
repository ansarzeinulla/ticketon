import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { SelectField, TextAreaField, TextField } from "./field";

/**
 * Form fields have to say *why* they are invalid, not just that they are
 * (SRS 7: "Event pages should meet WCAG 2.1 AA accessibility guidelines").
 *
 * The association was broken until now: aria-errormessage pointed at an id
 * that was never rendered, so a screen reader announced the field as invalid
 * and then had nothing to read out. An attribute referencing a missing element
 * is silently ignored, which is exactly the kind of failure that never shows
 * up in a visual check - hence a test.
 */

describe("TextField", () => {
  it("labels the control", () => {
    render(<TextField label="Email" name="email" />);
    expect(screen.getByLabelText(/Email/)).toBeInTheDocument();
  });

  it("marks itself invalid and points at the message that says why", () => {
    render(<TextField label="Email" name="email" error="Enter a valid email address." />);

    const input = screen.getByLabelText(/Email/);
    expect(input).toHaveAttribute("aria-invalid", "true");

    const errorID = input.getAttribute("aria-errormessage");
    expect(errorID).toBeTruthy();

    // The element it names has to exist, or the reference is inert.
    const message = document.getElementById(errorID as string);
    expect(message).not.toBeNull();
    expect(message).toHaveTextContent("Enter a valid email address.");
    expect(message).toHaveAttribute("role", "alert");
  });

  it("carries no invalid state when there is no error", () => {
    render(<TextField label="Email" name="email" hint="We never share it." />);
    const input = screen.getByLabelText(/Email/);
    expect(input).not.toHaveAttribute("aria-invalid");
    expect(input).not.toHaveAttribute("aria-errormessage");
    expect(screen.getByText("We never share it.")).toBeInTheDocument();
  });

  it("shows the error instead of the hint, rather than both", () => {
    render(
      <TextField label="Email" name="email" hint="We never share it." error="Required." />,
    );
    expect(screen.getByText("Required.")).toBeInTheDocument();
    expect(screen.queryByText("We never share it.")).not.toBeInTheDocument();
  });

  it("marks a required field for sighted and assistive users alike", () => {
    render(<TextField label="Email" name="email" required />);
    expect(screen.getByLabelText(/Email/)).toBeRequired();
  });
});

describe("SelectField", () => {
  it("associates its error message too", () => {
    render(
      <SelectField
        label="Visibility"
        name="visibility"
        options={["public", "private"]}
        error="Choose one."
      />,
    );

    const select = screen.getByLabelText(/Visibility/);
    expect(select).toHaveAttribute("aria-invalid", "true");

    const errorID = select.getAttribute("aria-errormessage");
    expect(document.getElementById(errorID as string)).toHaveTextContent("Choose one.");
  });
});

describe("TextAreaField", () => {
  it("associates its error message too", () => {
    render(
      <TextAreaField
        label="Description"
        name="description"
        value=""
        onChange={() => {}}
        error="Too long."
      />,
    );

    const textarea = screen.getByLabelText(/Description/);
    expect(textarea).toHaveAttribute("aria-invalid", "true");

    const errorID = textarea.getAttribute("aria-errormessage");
    expect(document.getElementById(errorID as string)).toHaveTextContent("Too long.");
  });
});

describe("every field type", () => {
  it("gives each instance its own ids, so two on a page do not collide", () => {
    render(
      <>
        <TextField label="Contact email" name="contact_email" error="Bad." />
        <TextField label="Sign-in email" name="email" error="Also bad." />
      </>,
    );

    const first = screen.getByLabelText(/Contact email/).getAttribute("aria-errormessage");
    const second = screen.getByLabelText(/Sign-in email/).getAttribute("aria-errormessage");

    expect(first).not.toBe(second);
    expect(document.getElementById(first as string)).toHaveTextContent("Bad.");
    expect(document.getElementById(second as string)).toHaveTextContent("Also bad.");
  });
});
