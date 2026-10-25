import Link from "next/link";
import {
  ArrowDown,
  ArrowRight,
  Check,
  CornerDownRight,
  MoveUpRight,
} from "lucide-react";
import { CreateLinkForm } from "@/components/create-link-form";

export default function Home() {
  return (
    <>
      <section className="hero">
        <div className="hero-copy">
          <p className="eyebrow">
            <span />
            LESS URL. MORE POSSIBILITY.
          </p>
          <h1>
            A little link.
            <br />A long way.
          </h1>
          <p className="hero-description">
            Good things deserve simple links. Turn long, winding URLs into
            something short, memorable, and ready to share.
          </p>
          <div className="hero-benefits">
            <span>
              <Check size={15} aria-hidden="true" />
              Make it memorable
            </span>
            <span>
              <Check size={15} aria-hidden="true" />
              Keep it in your control
            </span>
          </div>
          <Link className="text-link" href="/dashboard/">
            Visit your link library <ArrowRight size={16} aria-hidden="true" />
          </Link>
        </div>
        <div className="link-illustration" aria-hidden="true">
          <span className="illustration-label">
            A SMALL CHANGE, A BETTER CONNECTION
          </span>
          <div className="long-url">
            https://example.com/a/very/long/destination
          </div>
          <ArrowDown
            className="illustration-arrow"
            size={29}
            strokeWidth={1.4}
          />
          <div className="short-ticket">
            <span className="ticket-dot" />
            shortly<span>/</span>your-story
            <MoveUpRight size={23} />
          </div>
          <span className="illustration-caption">
            <CornerDownRight size={15} />
            Same destination. A simpler journey.
          </span>
          <span className="orbit orbit-one" />
          <span className="orbit orbit-two" />
        </div>
      </section>
      <CreateLinkForm />
      <section className="how-it-works" aria-label="How it works">
        {[
          {
            n: "01",
            title: "Start with a destination",
            text: "An article, a project, a thing you love. Any HTTP or HTTPS link works.",
          },
          {
            n: "02",
            title: "Make it your own",
            text: "Choose a memorable alias, add an expiration, or keep things simple.",
          },
          {
            n: "03",
            title: "Send it on its way",
            text: "Copy your link, share it anywhere, and manage it from your library.",
          },
        ].map((step) => (
          <div key={step.n}>
            <span>{step.n}</span>
            <h2>{step.title}</h2>
            <p>{step.text}</p>
          </div>
        ))}
      </section>
    </>
  );
}
