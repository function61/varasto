import { DangerAlert } from 'f61ui/component/alerts';
import { Loading } from 'f61ui/component/loading';
import {
	coerceToStructuredErrorResponse,
	formatStructuredErrorResponse,
	handleKnownGlobalErrors,
} from 'f61ui/errors';
import { shouldAlwaysSucceed } from 'f61ui/utils';
import * as React from 'react';

// wraps result data for UI friendly and typesafe way, acknowledging that the result can
// be in these states:
//
//   a) loading
//   b) fetched succcessfully
//   c) errored fetching the value
//
export class Result<T> {
	static unwrap2<T1, T2>(
		a: Result<T1>,
		b: Result<T2>,
	): [T1 | undefined, T2 | undefined, React.ReactNode] {
		const [resA, loadingOrErrorA] = a.unwrap();
		const [resB, loadingOrErrorB] = b.unwrap();

		// TODO: if b errored and a is loading, we should prioritize and display the error message?
		// or the other way around? anyways, currently we're doing "first wins" and it's not good
		return [resA, resB, loadingOrErrorA || loadingOrErrorB];
	}

	static unwrap3<T1, T2, T3>(
		a: Result<T1>,
		b: Result<T2>,
		c: Result<T3>,
	): [T1 | undefined, T2 | undefined, T3 | undefined, React.ReactNode] {
		const [resA, loadingOrErrorA] = a.unwrap();
		const [resB, loadingOrErrorB] = b.unwrap();
		const [resC, loadingOrErrorC] = c.unwrap();

		return [resA, resB, resC, loadingOrErrorA || loadingOrErrorB || loadingOrErrorC];
	}

	private result?: T;
	private loading = false;
	private errorText = '';
	private keepResult = false;
	private change: (result: Result<T>) => void;

	// takes in a callback that is called when we transition to loading, error or data fetch success
	constructor(change: (result: Result<T>) => void) {
		this.change = change;
	}

	draw(fn: (obj: T) => React.ReactNode) {
		const [obj2, loadingOrError] = this.unwrap();

		if (loadingOrError || obj2 === undefined) {
			return loadingOrError;
		}

		return fn(obj2);
	}

	unwrap(): [T | undefined, React.ReactNode] {
		const loadingOrError = this.loading ? (
			<Loading />
		) : this.errorText ? (
			<DangerAlert>{this.errorText}</DangerAlert>
		) : null;

		return [this.result, loadingOrError];
	}

	load(start: () => Promise<T>) {
		// shoudln't ever throw - errors are handled internally
		shouldAlwaysSucceed(this.loadInternal(start));
	}

	loadWhileKeepingOldResult(start: () => Promise<T>) {
		this.keepResult = true;

		this.load(start);
	}

	private async loadInternal(start: () => Promise<T>) {
		if (this.loading) {
			// ignore concurrent calls, since the last one is still busy
			return;
		}

		this.loading = true;
		this.errorText = '';

		// might not be the first load() call, so clear any previous data
		if (!this.keepResult) {
			this.result = undefined;
		}

		this.change(this);

		try {
			const prom = start();

			this.result = await prom;
		} catch (err) {
			// if "keepResult", we did so because we explicitly wanted to keep previously
			// loaded content present while the updated content is being fetched. now that
			// we encountered an exceptional case (error), we should clear old content so
			// be extra explicit to the user that this is not the new content
			this.result = undefined;

			const ser = coerceToStructuredErrorResponse(err);

			handleKnownGlobalErrors(ser);

			this.errorText = formatStructuredErrorResponse(ser);
		}

		this.loading = false;

		this.change(this);
	}
}
