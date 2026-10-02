import { DangerAlert } from 'f61ui/component/alerts';
import { Panel, tableClassStripedHover } from 'f61ui/component/bootstrap';
import { CommandButton } from 'f61ui/component/CommandButton';
import { Result } from 'f61ui/component/result';
import { volumesIntegrityUrl } from 'generated/frontend_uiroutes';
import { IntegrityverificationjobReverify } from 'generated/stoserver/stoservertypes_commands';
import { getIntegrityVerificationJobs } from 'generated/stoserver/stoservertypes_endpoints';
import { IntegrityVerificationJob } from 'generated/stoserver/stoservertypes_types';
import { AdminLayout } from 'layout/AdminLayout';
import * as React from 'react';

interface IntegrityVerificationJobPageProps {
	id: string;
}

interface IntegrityVerificationJobPageState {
	jobs: Result<IntegrityVerificationJob[]>;
	selectedIssueIndexes: number[];
}

export default class IntegrityVerificationJobPage extends React.Component<
	IntegrityVerificationJobPageProps,
	IntegrityVerificationJobPageState
> {
	state: IntegrityVerificationJobPageState = {
		jobs: new Result<IntegrityVerificationJob[]>((_) => {
			this.setState({ jobs: _ });
		}),
		selectedIssueIndexes: [],
	};

	componentDidMount() {
		this.loadJob();
	}

	componentWillReceiveProps() {
		this.setState({ selectedIssueIndexes: [] });
		this.loadJob();
	}

	render() {
		const [jobs, loadingOrError] = this.state.jobs.unwrap();

		return (
			<AdminLayout
				title="Integrity verification job"
				breadcrumbs={[{ url: volumesIntegrityUrl(), title: 'Integrity verification' }]}>
				<Panel heading="Integrity verification issues">
					{loadingOrError || (jobs && this.renderJob(jobs))}
				</Panel>
			</AdminLayout>
		);
	}

	private renderJob(jobs: IntegrityVerificationJob[]) {
		const job = jobs.find((candidate) => candidate.Id === this.props.id);
		if (!job) {
			return <DangerAlert>Integrity verification job not found.</DangerAlert>;
		}

		const selectableIssueIndexes = job.Issues.reduce<number[]>((indexes, issue, index) => {
			if (issue.Blob !== null) {
				indexes.push(index);
			}
			return indexes;
		}, []);
		const selectedCount = this.state.selectedIssueIndexes.length;
		const selectedBlobRefs: string[] = [];
		for (const index of this.state.selectedIssueIndexes) {
			const blob = job.Issues[index].Blob;
			if (blob !== null) {
				selectedBlobRefs.push(blob);
			}
		}

		return (
			<div>
				<table className={tableClassStripedHover}>
					<thead>
						<tr>
							<th style={{ width: '1%' }}>
								<input
									type="checkbox"
									disabled={selectableIssueIndexes.length === 0}
									onChange={() => {
										this.setState({
											selectedIssueIndexes: selectableIssueIndexes.filter(
												(index) =>
													this.state.selectedIssueIndexes.indexOf(
														index,
													) === -1,
											),
										});
									}}
									title="Invert blob issue selection"
								/>
							</th>
							<th>Blob</th>
							<th>Problem</th>
						</tr>
					</thead>
					<tbody>
						{job.Issues.map((issue, index) => (
							<tr key={index}>
								<td>
									{issue.Blob !== null && (
										<input
											type="checkbox"
											checked={
												this.state.selectedIssueIndexes.indexOf(index) !==
												-1
											}
											onChange={(event) => {
												this.setIssueSelected(index, event.target.checked);
											}}
										/>
									)}
								</td>
								<td>{issue.Blob || '(general)'}</td>
								<td>{issue.Problem}</td>
							</tr>
						))}
					</tbody>
				</table>

				{selectedCount > 0 && (
					<CommandButton
						command={IntegrityverificationjobReverify(job.Id, selectedBlobRefs)}
					/>
				)}
			</div>
		);
	}

	private setIssueSelected(index: number, selected: boolean) {
		this.setState((state) => {
			const selectedIssueIndexes = state.selectedIssueIndexes.filter(
				(item) => item !== index,
			);
			if (selected) {
				selectedIssueIndexes.push(index);
			}
			return { selectedIssueIndexes };
		});
	}

	private loadJob() {
		this.state.jobs.load(() => getIntegrityVerificationJobs());
	}
}
