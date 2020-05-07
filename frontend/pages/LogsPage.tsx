import { RefreshButton } from 'component/refreshbutton';
import { Result } from 'f61ui/component/result';
import { Panel, tableClassStripedHover } from 'f61ui/component/bootstrap';
import { getLogs } from 'generated/stoserver/stoservertypes_endpoints';
import { AdminLayout } from 'layout/AdminLayout';
import * as React from 'react';

interface LogsPageState {
	logs: Result<string[]>;
}

export default class LogsPage extends React.Component<{}, LogsPageState> {
	state: LogsPageState = {
		logs: new Result<string[]>((_) => {
			this.setState({ logs: _ });
		}),
	};

	componentDidMount() {
		this.fetchData();
	}

	componentWillReceiveProps() {
		this.fetchData();
	}

	render() {
		const [logs, loadingOrError] = this.state.logs.unwrap();

		return (
			<AdminLayout title="Logs" breadcrumbs={[]}>
				<Panel heading="Logs">
					<RefreshButton
						refresh={() => {
							this.fetchData();
						}}
					/>

					<table className={tableClassStripedHover}>
						<thead>
							<tr>
								<th>Line</th>
							</tr>
						</thead>
						<tbody>
							{(logs || []).map((line) => (
								<tr>
									<td>{line}</td>
								</tr>
							))}
						</tbody>
						<tfoot>
							<tr>
								<td colSpan={99}>{loadingOrError}</td>
							</tr>
						</tfoot>
					</table>

					<RefreshButton
						refresh={() => {
							this.fetchData();
						}}
					/>
				</Panel>
			</AdminLayout>
		);
	}

	private fetchData() {
		this.state.logs.loadWhileKeepingOldResult(() => getLogs());
	}
}
