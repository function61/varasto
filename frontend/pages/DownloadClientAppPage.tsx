import { DocLink } from 'component/doclink';
import { DangerAlert, InfoAlert } from 'f61ui/component/alerts';
import { Glyphicon, Panel, tableClassStripedHover } from 'f61ui/component/bootstrap';
import { Info } from 'f61ui/component/info';
import { DocRef } from 'generated/stoserver/stoservertypes_types';
import { isDevVersion, version } from 'generated/version';
import { HelpLayout } from 'layout/HelpLayout';
import * as React from 'react';

export default class DownloadClientAppPage extends React.Component<{}, {}> {
	render() {
		return (
			<HelpLayout title="Download client app" breadcrumbs={[]}>
				<Panel
					heading={
						<div>
							Client apps for different platforms &nbsp;
							<Info text="This page you're viewing is Varasto's server UI. Varasto also has a client components that you can install on each of your devices to keep your content synchronized with Varasto server." />
						</div>
					}>
					{isDevVersion && (
						<DangerAlert>
							You're using dev version, so the download links are broken (they are
							version-specific links). Visit the GitHub page instead.
						</DangerAlert>
					)}
					<table className={tableClassStripedHover}>
						<thead>
							<tr>
								<th>Type</th>
								<th>OS</th>
								<th>Architecture</th>
								<th>Download</th>
							</tr>
						</thead>
						<tbody>
							<tr>
								<td title="PC">💻</td>
								<td>Windows</td>
								<td>x86 64-bit</td>
								<td>{assetLink('sto.exe')}</td>
							</tr>
							<tr>
								<td title="PC">💻</td>
								<td>Linux</td>
								<td>x86 64-bit</td>
								<td>{assetLink('sto_linux-amd64')}</td>
							</tr>
							<tr>
								<td title="Single-board computer / embedded">📟</td>
								<td>Linux</td>
								<td>ARM (Raspberry Pi etc.)</td>
								<td>{assetLink('sto_linux-arm')}</td>
							</tr>
							<tr>
								<td title="Mobile">📱</td>
								<td>Android</td>
								<td></td>
								<td>
									Coming soon{' '}
									<Info text="If you're feeling adventurous you can try Linux/ARM on Android from command line" />
								</td>
							</tr>
							<tr>
								<td title="PC">💻</td>
								<td>macOS</td>
								<td>x86 64-bit</td>
								<td>{assetLink('sto_darwin-amd64')}</td>
							</tr>
							<tr>
								<td title="Mobile">📱</td>
								<td>iOS</td>
								<td />
								<td>
									Might come later <Info text="Android is better" />
								</td>
							</tr>
						</tbody>
					</table>

					<InfoAlert>
						Once you have downloaded the client app, follow installation instructions:{' '}
						<DocLink doc={DocRef.DocsDataInterfacesClientIndexMd} />
					</InfoAlert>
				</Panel>
			</HelpLayout>
		);
	}
}

function assetLink(binaryName: string): React.ReactNode {
	return (
		<a
			className="btn btn-default"
			href={
				'https://github.com/function61/varasto/releases/download/' +
				version +
				'/' +
				binaryName
			}
			target="_blank">
			{binaryName}
			&nbsp;
			<Glyphicon icon="download-alt" />
		</a>
	);
}
