import { boot } from 'f61ui/appcontroller';
import { DangerAlert } from 'f61ui/component/alerts';
import { GlobalConfig } from 'f61ui/globalconfig';
import { AppDefaultLayout } from 'layout/appdefaultlayout';
import * as React from 'react';
import { router } from 'router';

export function main(appElement: HTMLElement, config: GlobalConfig): void {
	const notFoundPage = (
		<AppDefaultLayout title="404" breadcrumbs={[]}>
			<h1>404</h1>

			<DangerAlert>The page you were looking for is not found.</DangerAlert>
		</AppDefaultLayout>
	);

	boot(appElement, config, { router, notFoundPage });
}
