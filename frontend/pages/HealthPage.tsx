import { Panel } from 'f61ui/component/bootstrap';
import { Loading } from 'f61ui/component/loading';
import { shouldAlwaysSucceed, unrecognizedValue } from 'f61ui/utils';
import { getHealth } from 'generated/stoserver/stoservertypes_endpoints';
import { Health, HealthStatus } from 'generated/stoserver/stoservertypes_types';
import { SettingsLayout } from 'layout/settingslayout';
import * as React from 'react';

interface HealthPageState {
	health?: Health;
}

export default class HealthPage extends React.Component<{}, HealthPageState> {
	state: HealthPageState = {};

	componentDidMount() {
		shouldAlwaysSucceed(this.fetchData());
	}

	componentWillReceiveProps() {
		shouldAlwaysSucceed(this.fetchData());
	}

	render() {
		return (
			<SettingsLayout title="Health" breadcrumbs={[]}>
				<Panel heading="Health">{this.renderHealth()}</Panel>
			</SettingsLayout>
		);
	}

	private renderHealth() {
		const health = this.state.health;

		if (!health) {
			return <Loading />;
		}

		const rows: JSX.Element[] = [];

		const pushHealthNodeAsRow = (node: Health, indentLevel: number) => {
			rows.push(
				<tr>
					<td>{healthStatusToIcon(node.Health)}</td>
					<td style={{ paddingLeft: indentLevel * 32 + 'px' }}>{node.Title}</td>
					<td>{node.Details}</td>
				</tr>,
			);

			node.Children.forEach((childHealth) => {
				pushHealthNodeAsRow(childHealth, indentLevel + 1);
			});
		};

		pushHealthNodeAsRow(health, 0);

		return (
			<table className="table table-striped table-hover">
				<thead>
					<tr>
						<th />
						<th>Title</th>
						<th>Details</th>
					</tr>
				</thead>
				<tbody>{rows}</tbody>
			</table>
		);
	}

	private async fetchData() {
		const health = await getHealth();

		this.setState({ health });
	}
}

function healthStatusToIcon(input: HealthStatus): JSX.Element {
	switch (input) {
		case HealthStatus.Fail:
			return (
				<span className="alert alert-danger">
					<span className="glyphicon glyphicon-fire" />
				</span>
			);
		case HealthStatus.Warn:
			return (
				<span className="alert alert-warning">
					<span className="glyphicon glyphicon-warning-sign" />
				</span>
			);
		case HealthStatus.Pass:
			return (
				<span className="alert alert-success">
					<span className="glyphicon glyphicon-ok" />
				</span>
			);
		default:
			throw unrecognizedValue(input);
	}
}
