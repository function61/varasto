import { Panel } from 'f61ui/component/bootstrap';
import { Info } from 'f61ui/component/info';
import { version } from 'generated/version';
import { AppDefaultLayout } from 'layout/appdefaultlayout';
import * as React from 'react';

export default class DownloadClientAppPage extends React.Component<{}, {}> {
	render() {
		const title = 'Download client app';
		return (
			<AppDefaultLayout title={title} breadcrumbs={[]}>
				<Panel heading={title}>
					<table className="table table-striped table-hover">
						<thead>
							<tr>
								<th>Type</th>
								<th>OS</th>
								<th>Hardware</th>
								<th>Download</th>
							</tr>
						</thead>
						<tbody>
							<tr>
								<td title="PC">💻</td>
								<td>Windows</td>
								<td>64-bit x64</td>
								<td>{bintrayLink('sto.exe')}</td>
							</tr>
							<tr>
								<td title="PC">💻</td>
								<td>Linux</td>
								<td>64-bit x64</td>
								<td>{bintrayLink('sto_linux-amd64')}</td>
							</tr>
							<tr>
								<td />
								<td>Linux</td>
								<td>ARM (Raspberry Pi etc.)</td>
								<td>{bintrayLink('sto_linux-arm')}</td>
							</tr>
							<tr>
								<td title="Mobile">📱</td>
								<td>Android</td>
								<td>Any</td>
								<td>
									Coming soon{' '}
									<Info text="If you're feeling adventurous you can try Linux/ARM on Android from command line" />
								</td>
							</tr>
							<tr>
								<td title="PC">💻</td>
								<td>macOS</td>
								<td>64-bit x64</td>
								<td>{bintrayLink('sto_darwin-amd64')}</td>
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
				</Panel>
			</AppDefaultLayout>
		);
	}
}

function bintrayLink(binaryName: string): React.ReactNode {
	return (
		<a
			href={
				'https://bintray.com/function61/dl/download_file?file_path=varasto%2F' +
				version +
				'%2F' +
				binaryName
			}
			target="_blank">
			Download
		</a>
	);
}
