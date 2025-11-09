import { Sensitivity } from 'component/sensitivity';
import { Glyphicon, Panel, tableClassStripedHover } from 'f61ui/component/bootstrap';
import { Result } from 'f61ui/component/result';
import { browseUrl, collectionUrl, searchUrl } from 'generated/frontend_uiroutes';
import { search } from 'generated/stoserver/stoservertypes_endpoints';
import { SearchResult } from 'generated/stoserver/stoservertypes_types';
import { AppDefaultLayout } from 'layout/appdefaultlayout';
import * as React from 'react';

interface SearchPageProps {
	query: string;
}

interface SearchPageState {
	queryTransient: string;
	results: Result<SearchResult[]>;
}

export default class SearchPage extends React.Component<SearchPageProps, SearchPageState> {
	state: SearchPageState = {
		queryTransient: this.props.query,
		results: new Result<SearchResult[]>((_) => {
			this.setState({ results: _ });
		}),
	};

	componentDidMount() {
		this.fetchData();
	}

	componentWillReceiveProps() {
		this.fetchData();
	}

	render() {
		return (
			<AppDefaultLayout title="Search" breadcrumbs={[]}>
				<Panel heading="Search">{this.renderData()}</Panel>
			</AppDefaultLayout>
		);
	}

	private renderData() {
		const [results, loadingOrError] = this.state.results.unwrap();

		const resultsFiltered = (results || []).filter((result) => {
			if (result.Collection && result.Collection.Sensitivity !== Sensitivity.FamilyFriendly) {
				return false;
			}

			if (
				result.Directory &&
				result.Directory.Directory.Directory.Sensitivity !== Sensitivity.FamilyFriendly
			) {
				return false;
			}

			return true;
		});

		return (
			<div>
				<form action={searchUrl({ q: '' })} method="get">
					<div className="input-group">
						<input
							className="form-control"
							name="q"
							value={this.state.queryTransient}
							onChange={(e) => {
								this.setState({ queryTransient: e.target.value });
							}}
							autoFocus={true}
						/>
						<span className="input-group-btn">
							<button className="btn btn-default" type="submit">
								🔍
							</button>
						</span>
					</div>
				</form>

				<table className={tableClassStripedHover}>
					<thead>
						<tr>
							<th>Kind</th>
							<th>Result</th>
						</tr>
					</thead>
					<tbody>
						{resultsFiltered.map((result: SearchResult) => {
							const kindIndicator = ((): [string, JSX.Element, JSX.Element] => {
								if (result.Collection) {
									return [
										`coll-${result.Collection.Id}`,
										<Glyphicon icon="duplicate" />,
										<a href={collectionUrl({ id: result.Collection.Id })}>
											{result.Collection.Name}
										</a>,
									];
								}
								if (result.Directory) {
									return [
										`dir-${result.Directory.Directory.Directory.Id}`,
										<Glyphicon icon="folder-open" />,
										<a
											href={browseUrl({
												dir: result.Directory.Directory.Directory.Id,
											})}>
											{result.Directory.Directory.Directory.Name}
										</a>,
									];
								}
								throw new Error('should not happen');
							})();

							return (
								<tr id={kindIndicator[0]}>
									<td>{kindIndicator[1]}</td>
									<td>{kindIndicator[2]}</td>
								</tr>
							);
						})}
					</tbody>
					<tfoot>{loadingOrError}</tfoot>
				</table>
			</div>
		);
	}

	private fetchData() {
		this.state.results.load(() => search(this.props.query));
	}
}
