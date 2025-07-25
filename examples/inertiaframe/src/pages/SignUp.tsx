import { Field } from "@base-ui-components/react/field";
import { Form } from "@base-ui-components/react/form";
import { Link, useForm } from "@inertiajs/react";

export default function SignUp() {
  const form = useForm({
    email: "",
    password: "",
  });

  return (
    <main>
      <h1>Sign Up</h1>
      <Form
        aria-disabled={form.processing}
        onSubmit={async (event) => {
          event.preventDefault();
          form.post("/sign-up", {});
        }}
      >
        <Field.Root>
          <Field.Label>Email</Field.Label>
          <Field.Control
            required
            type="email"
            name="email"
            placeholder="you@email.com"
            value={form.data.email}
            onChange={(e) => form.setData("email", e.target.value)}
          />
          <Field.Error match="valueMissing">Please enter your name</Field.Error>
          <Field.Error match={!!form.errors.email}>
            {form.errors.email}
          </Field.Error>
        </Field.Root>
        <Field.Root>
          <Field.Label>Password</Field.Label>
          <Field.Control
            // required
            type="password"
            name="password"
            placeholder="my very secure password"
            value={form.data.password}
            onChange={(e) => form.setData("password", e.target.value)}
          />
          <Field.Error match="valueMissing">
            Please enter your password
          </Field.Error>
          <Field.Error match={!!form.errors.password}>
            {form.errors.password}
          </Field.Error>
        </Field.Root>
        <button disabled={form.processing}>Submit</button>
      </Form>
      <Link href="/sign-in">Sign In</Link>
    </main>
  );
}
